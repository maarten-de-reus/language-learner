// Het overzicht van alle kaarten voor het beheer, met per kaart wat erbij hoort.

package catalog

import (
	"slices"
	"strings"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
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
	Verified   bool      `json:"verified"`             // goedgekeurd door de controleur
	ReviewNote string    `json:"reviewNote,omitempty"` // opmerking van de controleur
	Priority   bool      `json:"priority"`             // voorrang in de wachtrij (gemarkeerd in Kaarten)
	Attempts   int       `json:"attempts"`             // eerste pogingen volgens het antwoordenlog (ingevuld door de API)
	Wrong      int       `json:"wrong"`                // waarvan fout
	Step       int       `json:"step"`
	Interval   int       `json:"interval"` // dagen bij de huidige stap
	NextReview time.Time `json:"nextReview"`
}

// Entries geeft alle concepten, gesorteerd op bronwoord (accenten genegeerd) en daarna hint
func Entries(lib *storage.Library) []Entry {
	all := make([]int, len(lib.Concepts))
	for i := range all {
		all[i] = i
	}
	return entriesOf(lib, all)
}

// entriesOf maakt de Entry's van de concepten op deze posities, gesorteerd zoals Entries.
// De omgekeerde richtingen worden via een index op bronwoord gezocht: per concept alle andere
// concepten doorlopen kost bij een import met veel keuzes al snel seconden.
func entriesOf(lib *storage.Library, idx []int) []Entry {
	meanings := map[string]int{}
	bySource := map[string][]int{}
	for i, c := range lib.Concepts {
		meanings[meaningKey(lib, c)]++
		bySource[c.SourceWordID] = append(bySource[c.SourceWordID], i)
	}
	reverse := func(c storage.TranslationConcept) []string {
		var ids []string
		for _, t := range c.TargetWordIDs {
			for _, j := range bySource[t] {
				if o := lib.Concepts[j]; o.ID != c.ID && slices.Contains(o.TargetWordIDs, c.SourceWordID) && !slices.Contains(ids, o.ID) {
					ids = append(ids, o.ID)
				}
			}
		}
		return ids
	}

	entries := make([]Entry, 0, len(idx))
	for _, i := range idx {
		c := lib.Concepts[i]
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
			Reverse:    reverse(c),
			New:        c.FirstReview.IsZero(),
			Groups:     append([]string{}, c.Groups...),
			Verified:   c.Verified,
			ReviewNote: c.ReviewNote,
			Priority:   c.Priority,
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
