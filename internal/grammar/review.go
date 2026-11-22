// Regels in de herhalingen: elke regel heeft een planning zoals een woord. Een herhaling is een
// paar opgaven van die regel; bijna allemaal goed = een stap verder, anders terug.

package grammar

import (
	"slices"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

const (
	ReviewSize  = 5              // opgaven per regel in een herhaling
	PassCorrect = 4              // zoveel goed om een stap verder te gaan
	RulesPerDay = 5              // zoveel regels per dag in de herhalingen
	maxRuleStep = engine.TopStep // regels gaan niet verder dan 90 dagen (geen 'beheerst')
)

// DueRule is een regel die aan de beurt is
type DueRule struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
	Step  int    `json:"step"`
	New   bool   `json:"new"` // nog nooit herhaald
}

// DueRules: de actieve regels die klaar zijn en aan de beurt (of nieuw), te laat eerst, nieuwe
// als laatste, tot RulesPerDay min wat er vandaag al gedaan is
func DueRules(words []Word, active []string, progress map[string]storage.RuleProgress, now time.Time) (due []DueRule, doneToday int) {
	today := engine.DayOf(now)
	for _, p := range progress {
		if !p.LastReview.IsZero() && engine.DayOf(p.LastReview).Equal(today) {
			doneToday++
		}
	}
	for _, r := range Rules {
		if !slices.Contains(active, r.ID) {
			continue
		}
		if _, ready := r.Ready(words); !ready {
			continue
		}
		p, seen := progress[r.ID]
		if seen && p.NextReview.After(now) {
			continue
		}
		due = append(due, DueRule{ID: r.ID, Name: r.Name, Group: r.Group, Step: p.Step, New: !seen})
	}
	slices.SortStableFunc(due, func(a, b DueRule) int {
		switch {
		case a.New != b.New:
			if a.New {
				return 1
			}
			return -1
		case !a.New:
			return progress[a.ID].NextReview.Compare(progress[b.ID].NextReview)
		}
		return 0
	})
	if left := RulesPerDay - doneToday; len(due) > max(left, 0) {
		due = due[:max(left, 0)]
	}
	return due, doneToday
}

// ApplyReview verwerkt een herhaling van een regel: PassCorrect of meer goed = een stap verder,
// anders twee stappen terug en morgen opnieuw (net als een vergeten woord)
func ApplyReview(p storage.RuleProgress, correct int, now time.Time) (storage.RuleProgress, bool) {
	passed := correct >= PassCorrect
	if p.FirstReview.IsZero() {
		p.FirstReview = now
		p.Step = -1 // de eerste geslaagde herhaling brengt de regel op stap 0 (morgen weer)
	}
	if passed {
		p.Step = min(p.Step+1, maxRuleStep)
		p.NextReview = engine.ReviewDate(now, engine.AnkiIntervals[p.Step])
	} else {
		p.Step = max(p.Step-engine.LapseSteps, 0)
		p.NextReview = engine.ReviewDate(now, 1)
	}
	p.LastReview = now
	return p, passed
}
