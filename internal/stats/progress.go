package stats

import (
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

// ProgressDays is hoe ver de grafiek "Voortgang door de tijd" terugkijkt
const ProgressDays = 90

// ProgressDay is hoeveel geoefende kaarten er aan het eind van een dag op elke stap stonden
type ProgressDay struct {
	Date  string                         `json:"date"` // JJJJ-MM-DD
	Steps [len(engine.AnkiIntervals)]int `json:"steps"`
}

// progress rekent uit het antwoordenlog terug hoe je kaarten per dag over de stappen verdeeld
// waren: na elk antwoord staat een kaart op de stap die in het log staat. Een kaart die al
// geoefend was voordat het log begon, staat vanaf de eerste dag op zijn stap van vóór zijn eerste
// antwoord in het log. Nieuwe kaarten tellen pas mee na hun eerste antwoord, verwijderde nooit.
// reviews moeten op tijd gesorteerd zijn en Toch goed al verwerkt hebben (applyOverrides).
func progress(lib *storage.Library, reviews []storage.Review, today time.Time) []ProgressDay {
	if len(reviews) == 0 {
		return []ProgressDay{}
	}
	loc := today.Location()
	live := map[string]bool{}
	for _, c := range lib.Concepts {
		live[c.ID] = true
	}
	step := map[string]int{}
	seen := map[string]bool{}
	for _, r := range reviews {
		if !seen[r.ConceptID] && live[r.ConceptID] && r.StepBefore > 0 {
			step[r.ConceptID] = r.StepBefore // al geoefend voor het log begon
		}
		seen[r.ConceptID] = true
	}

	first := engine.DayOf(reviews[0].Time.In(loc))
	if start := today.AddDate(0, 0, -(ProgressDays - 1)); first.Before(start) {
		first = start
	}
	var out []ProgressDay
	i := 0
	for day := engine.DayOf(reviews[0].Time.In(loc)); !day.After(today); day = day.AddDate(0, 0, 1) {
		for ; i < len(reviews) && !engine.DayOf(reviews[i].Time.In(loc)).After(day); i++ {
			if r := reviews[i]; live[r.ConceptID] {
				step[r.ConceptID] = min(max(r.StepAfter, 0), len(engine.AnkiIntervals)-1)
			}
		}
		if day.Before(first) {
			continue // te lang geleden voor de grafiek, maar de stand loopt wel mee
		}
		d := ProgressDay{Date: day.Format(time.DateOnly)}
		for _, s := range step {
			d.Steps[s]++
		}
		out = append(out, d)
	}
	return out
}
