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
	Meanings   int       `json:"meanings"` // aantal concepten met hetzelfde bronwoord en dezelfde richting
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
