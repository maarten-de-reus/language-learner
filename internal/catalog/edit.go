package catalog

import (
	"errors"
	"slices"
	"strings"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

var (
	ErrNotFound     = errors.New("concept niet gevonden")
	ErrNoAnswers    = errors.New("er moet minstens één goed antwoord zijn")
	ErrContextTaken = errors.New("een ander concept met hetzelfde woord en dezelfde richting heeft al deze hint; kies een hint die de betekenis onderscheidt")
)

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
		if a = engine.Clean(a); a != "" && !slices.ContainsFunc(answers, func(x string) bool { return strings.EqualFold(x, a) }) {
			answers = append(answers, a)
		}
	}
	if len(answers) == 0 {
		return ErrNoAnswers
	}

	src := lib.Words[c.SourceWordID]
	prompt := engine.Clean(e.Prompt)
	if prompt == "" {
		prompt = src.Text
	}

	context := engine.Clean(e.Context)
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

	// Is er inhoudelijk iets veranderd, dan moet het woord opnieuw gecontroleerd worden
	changed := prompt != src.Text || context != c.ContextSource || example != c.Example ||
		!slices.Equal(answers, TargetTexts(lib, c))
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
	if changed {
		for _, id := range append([]string{c.ID}, reverseIDs(lib, c)...) {
			lib.Concepts[lib.Index(id)].Verified = false
		}
	}
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

// reverseIDs geeft de ID's van de omgekeerde richting(en) van c (zie storage.Reverses)
func reverseIDs(lib *storage.Library, c storage.TranslationConcept) []string {
	return idsAt(lib, storage.NewReverses(lib.Concepts).Of(c))
}

func idsAt(lib *storage.Library, idx []int) []string {
	ids := make([]string, 0, len(idx))
	for _, i := range idx {
		ids = append(ids, lib.Concepts[i].ID)
	}
	return ids
}
