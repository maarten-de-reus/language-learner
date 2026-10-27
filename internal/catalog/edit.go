package catalog

import (
	"errors"
	"slices"
	"strings"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

var (
	ErrNotFound     = errors.New("concept niet gevonden")
	ErrNoAnswers    = errors.New("er moet minstens één goed antwoord zijn")
	ErrContextTaken = errors.New("een ander concept met hetzelfde woord en dezelfde richting heeft al deze hint; kies een hint die de betekenis onderscheidt")
)

// Entry is één concept zoals het beheer het toont
type Entry struct {
	ID         string    `json:"id"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	Prompt     string    `json:"prompt"`
	Context    string    `json:"context"`
	Answers    []string  `json:"answers"`
	Example    string    `json:"example,omitempty"`
	Meanings   int       `json:"meanings"`          // aantal concepten met hetzelfde bronwoord en dezelfde richting
	Reverse    []string  `json:"reverse,omitempty"` // ID's van de omgekeerde richting(en), bijv. kávé -> koffie bij koffie -> kávé
	New        bool      `json:"new"`               // nog nooit gevraagd: staat in de wachtrij
	Groups     []string  `json:"groups"`
	Attempts   int       `json:"attempts"` // eerste pogingen volgens het antwoordenlog (ingevuld door de API)
	Wrong      int       `json:"wrong"`    // waarvan fout
	Step       int       `json:"step"`
	Interval   int       `json:"interval"` // dagen bij de huidige stap
	NextReview time.Time `json:"nextReview"`
}

// Entries geeft alle concepten, gesorteerd op bronwoord (accenten genegeerd) en daarna hint
func Entries(lib *storage.Library) []Entry {
	meanings := map[string]int{}
	for _, c := range lib.Concepts {
		meanings[meaningKey(lib, c)]++
	}

	entries := make([]Entry, 0, len(lib.Concepts))
	for _, c := range lib.Concepts {
		src := lib.Words[c.SourceWordID]
		e := Entry{
			ID:         c.ID,
			From:       src.Language,
			To:         targetLang(lib, c),
			Prompt:     src.Text,
			Context:    c.ContextSource,
			Answers:    TargetTexts(lib, c),
			Example:    c.Example,
			Meanings:   meanings[meaningKey(lib, c)],
			Reverse:    reverseIDs(lib, c),
			New:        c.FirstReview.IsZero(),
			Groups:     append([]string{}, c.Groups...),
			Step:       c.StepIndex,
			NextReview: c.NextReview,
		}
		if c.StepIndex >= 0 && c.StepIndex < len(engine.AnkiIntervals) {
			e.Interval = engine.AnkiIntervals[c.StepIndex]
		}
		entries = append(entries, e)
	}
	slices.SortStableFunc(entries, func(a, b Entry) int {
		if c := strings.Compare(engine.Fold(a.Prompt), engine.Fold(b.Prompt)); c != 0 {
			return c
		}
		return strings.Compare(engine.Fold(a.Context), engine.Fold(b.Context))
	})
	return entries
}

// Edit zijn de velden die je van een bestaand concept kunt aanpassen
type Edit struct {
	// Prompt is het woord dat vertaald moet worden (leeg = ongewijzigd). Het woord kan gedeeld zijn
	// met andere betekenissen (idő ☀️ / idő ⏰): alleen dit concept en zijn omgekeerde richting
	// gaan naar de nieuwe tekst, de andere betekenissen houden het oude woord.
	Prompt  string   `json:"prompt"`
	Context string   `json:"context"`
	Answers []string `json:"answers"`
	Example string   `json:"example"`
	// Groups vervangt de groepen (nil = ongewijzigd). Wat erbij komt of eraf gaat, geldt ook
	// voor de omgekeerde richting: een groep hoort bij het woord, niet bij één richting.
	Groups []string `json:"groups"`
}

// Update past een concept aan. Alle controles gebeuren vóór er iets aangepast wordt.
func Update(lib *storage.Library, id string, e Edit) error {
	i := lib.Index(id)
	if i < 0 {
		return ErrNotFound
	}
	c := lib.Concepts[i]

	var answers []string
	for _, a := range e.Answers {
		if a = strings.TrimSpace(a); a != "" && !slices.ContainsFunc(answers, func(x string) bool { return strings.EqualFold(x, a) }) {
			answers = append(answers, a)
		}
	}
	if len(answers) == 0 {
		return ErrNoAnswers
	}

	src := lib.Words[c.SourceWordID]
	prompt := strings.TrimSpace(e.Prompt)
	if prompt == "" {
		prompt = src.Text
	}

	context := strings.TrimSpace(e.Context)
	key := src.Language + "|" + targetLang(lib, c) + "|" + strings.ToLower(prompt)
	for j, other := range lib.Concepts {
		if j != i && meaningKey(lib, other) == key && strings.EqualFold(strings.TrimSpace(other.ContextSource), context) {
			return ErrContextTaken
		}
	}

	lang := targetLang(lib, c)
	var targetIDs []string
	for _, a := range answers {
		// Een antwoord dat al bij dit concept hoort houdt zijn eigen woord. Alleen nieuwe antwoorden
		// worden op tekst opgezocht: bij twee woorden met dezelfde tekst (bijv. "zij") zou dat
		// anders ongemerkt naar het andere woord kunnen springen.
		id := ""
		for _, t := range c.TargetWordIDs {
			if strings.EqualFold(strings.TrimSpace(lib.Words[t].Text), a) {
				id = t
				break
			}
		}
		if id == "" {
			id = ensureWord(lib.Words, lang, a)
		}
		if !slices.Contains(targetIDs, id) {
			targetIDs = append(targetIDs, id)
		}
	}

	var groups []string
	if e.Groups != nil {
		var err error
		if groups, err = NormalizeGroups(lib, e.Groups); err != nil {
			return err
		}
	}

	example := strings.TrimSpace(e.Example)
	exampleLanguage := c.ExampleLang
	if example == "" {
		exampleLanguage = ""
	} else if exampleLanguage == "" {
		exampleLanguage = exampleLang(example)
	}

	if prompt != src.Text {
		retarget(lib, &c, prompt)
	}
	c.ContextSource = context
	c.TargetWordIDs = targetIDs
	c.Example, c.ExampleLang = example, exampleLanguage
	if e.Groups != nil {
		var added, removed []string
		for _, g := range groups {
			if !slices.Contains(c.Groups, g) {
				added = append(added, g)
			}
		}
		for _, g := range c.Groups {
			if !slices.Contains(groups, g) {
				removed = append(removed, g)
			}
		}
		for _, rid := range reverseIDs(lib, c) {
			r := &lib.Concepts[lib.Index(rid)]
			addGroups(r, added)
			removeGroups(r, removed)
		}
		c.Groups = nil
		addGroups(&c, groups)
	}
	lib.Concepts[i] = c
	removeUnusedWords(lib)
	return nil
}

// retarget geeft c een bronwoord met de tekst prompt. Alleen hoofdletters anders: dan wordt het
// woord zelf aangepast. Anders gaat c naar een (bestaand of nieuw) woord met die tekst, en krijgt
// de omgekeerde richting dat woord als antwoord in plaats van het oude.
func retarget(lib *storage.Library, c *storage.TranslationConcept, prompt string) {
	old := lib.Words[c.SourceWordID]
	if strings.EqualFold(strings.TrimSpace(old.Text), prompt) {
		old.Text = prompt
		lib.Words[old.ID] = old
		return
	}
	reverse := reverseIDs(lib, *c)
	c.SourceWordID = ensureWord(lib.Words, old.Language, prompt)
	for _, rid := range reverse {
		r := &lib.Concepts[lib.Index(rid)]
		var targets []string
		for _, t := range r.TargetWordIDs {
			if t == old.ID {
				t = c.SourceWordID
			}
			if !slices.Contains(targets, t) {
				targets = append(targets, t)
			}
		}
		r.TargetWordIDs = targets
	}
}

// meaningKey groepeert concepten met hetzelfde bronwoord (taal + tekst) in dezelfde richting
func meaningKey(lib *storage.Library, c storage.TranslationConcept) string {
	src := lib.Words[c.SourceWordID]
	return src.Language + "|" + targetLang(lib, c) + "|" + strings.ToLower(strings.TrimSpace(src.Text))
}

func targetLang(lib *storage.Library, c storage.TranslationConcept) string {
	if len(c.TargetWordIDs) == 0 {
		return ""
	}
	return lib.Words[c.TargetWordIDs[0]].Language
}

// DeleteResult beschrijft wat Delete heeft gedaan
type DeleteResult struct {
	Deleted []string `json:"deleted"` // verwijderde concepten
	Updated []string `json:"updated"` // omgekeerde concepten waar alleen een antwoord uit is gehaald
}

// Delete verwijdert een concept, en met withReverse ook de omgekeerde richting.
// Een omgekeerd concept dat ook bij andere woorden hoort (szia -> hallo, hoi) wordt niet verwijderd;
// alleen dit woord gaat eruit als goed antwoord. Woorden die nergens meer gebruikt worden verdwijnen.
// Verwijderde concepten moeten daarna ook uit de lopende quiz (quiz.Session.Forget).
func Delete(lib *storage.Library, id string, withReverse bool) (DeleteResult, error) {
	var res DeleteResult
	i := lib.Index(id)
	if i < 0 {
		return res, ErrNotFound
	}
	c := lib.Concepts[i]
	res.Deleted = append(res.Deleted, id)

	if withReverse {
		for _, revID := range reverseIDs(lib, c) {
			r := &lib.Concepts[lib.Index(revID)]
			if remaining := slices.DeleteFunc(slices.Clone(r.TargetWordIDs), func(w string) bool { return w == c.SourceWordID }); len(remaining) > 0 {
				r.TargetWordIDs = remaining
				res.Updated = append(res.Updated, revID)
			} else {
				res.Deleted = append(res.Deleted, revID)
			}
		}
	}

	lib.Concepts = slices.DeleteFunc(lib.Concepts, func(x storage.TranslationConcept) bool { return slices.Contains(res.Deleted, x.ID) })
	removeUnusedWords(lib)
	return res, nil
}

// reverseIDs zoekt de concepten die de omgekeerde richting van c zijn: hun bronwoord is een
// antwoord van c, en het bronwoord van c is een van hun antwoorden
func reverseIDs(lib *storage.Library, c storage.TranslationConcept) []string {
	var ids []string
	for _, o := range lib.Concepts {
		if o.ID != c.ID && slices.Contains(c.TargetWordIDs, o.SourceWordID) && slices.Contains(o.TargetWordIDs, c.SourceWordID) {
			ids = append(ids, o.ID)
		}
	}
	return ids
}

func removeUnusedWords(lib *storage.Library) {
	used := map[string]bool{}
	for _, c := range lib.Concepts {
		used[c.SourceWordID] = true
		for _, t := range c.TargetWordIDs {
			used[t] = true
		}
	}
	for id := range lib.Words {
		if !used[id] {
			delete(lib.Words, id)
		}
	}
}
