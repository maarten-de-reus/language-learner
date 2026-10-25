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
	Attempts   int       `json:"attempts"`          // eerste pogingen volgens het antwoordenlog (ingevuld door de API)
	Wrong      int       `json:"wrong"`             // waarvan fout
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

// Edit zijn de velden die je van een bestaand concept kunt aanpassen.
// Het bronwoord zelf niet: dat kan door meerdere concepten gedeeld worden.
type Edit struct {
	Context string   `json:"context"`
	Answers []string `json:"answers"`
	Example string   `json:"example"`
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

	context := strings.TrimSpace(e.Context)
	key := meaningKey(lib, c)
	for j, other := range lib.Concepts {
		if j != i && meaningKey(lib, other) == key && strings.EqualFold(strings.TrimSpace(other.ContextSource), context) {
			return ErrContextTaken
		}
	}

	lang := targetLang(lib, c)
	var targetIDs []string
	for _, a := range answers {
		if id := ensureWord(lib.Words, lang, a); !slices.Contains(targetIDs, id) {
			targetIDs = append(targetIDs, id)
		}
	}

	example := strings.TrimSpace(e.Example)
	exampleLanguage := c.ExampleLang
	if example == "" {
		exampleLanguage = ""
	} else if exampleLanguage == "" {
		exampleLanguage = exampleLang(example)
	}

	c.ContextSource = context
	c.TargetWordIDs = targetIDs
	c.Example, c.ExampleLang = example, exampleLanguage
	lib.Concepts[i] = c
	return nil
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
// Een omgekeerd concept dat ook bij andere woorden hoort (ő -> hij, zij) wordt niet verwijderd;
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
