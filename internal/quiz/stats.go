// Tellers voor de kop en het eindscherm: wat er vandaag nog kan en hoe de woorden verdeeld zijn.

package quiz

import (
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

// Stats geeft een overzicht van alle concepten
type Stats struct {
	Mode      engine.Mode                    `json:"mode"` // herhalen of nieuwe woorden leren
	Total     int                            `json:"total"`
	Due       int                            `json:"due"`    // herhalingen die vandaag aan de beurt zijn
	DueNew    int                            `json:"dueNew"` // nieuwe woorden die je vandaag nog kunt leren
	Steps     [len(engine.AnkiIntervals)]int `json:"steps"`
	Intervals [len(engine.AnkiIntervals)]int `json:"intervals"`
	NextDue   *time.Time                     `json:"nextDue"`

	// Nieuwe woorden: wachtrij en dagelijkse limiet
	NewQueue int `json:"newQueue"` // nieuwe woorden die nog nooit gevraagd zijn
	// woorden in de wachtrij die nog niet goedgekeurd zijn: die komen pas in de les na controle.
	// Een woord telt één keer, ook al heeft het twee richtingen (zoals op het tabblad Controleren).
	NewUnverified int `json:"newUnverified"`
	NewToday      int `json:"newToday"`  // vandaag voor het eerst gevraagd
	NewPerDay     int `json:"newPerDay"` // ingestelde limiet; 0 = geen limiet
	NewLeft       int `json:"newLeft"`   // vandaag nog toegestaan, inclusief extra; -1 = geen limiet
	// Nieuwe woorden die vandaag nog zouden kunnen komen als de limiet het toeliet (zonder de
	// andere richting van woorden die vandaag al gevraagd zijn); bepaalt of "Nog 10" zin heeft
	NewAvailable int `json:"newAvailable"`
}

// unverifiedWords telt de woorden die de wachtrij blokkeren: met een nieuwe kaart die nog niet
// goedgekeurd is. De richtingen van een woord tellen samen één keer, zoals op het tabblad Controleren.
func (s *Session) unverifiedWords() int {
	reverses := storage.NewReverses(s.lib.Concepts)
	counted := map[string]bool{}
	n := 0
	for _, c := range s.lib.Concepts {
		if !engine.IsNew(c) || c.Verified || counted[c.ID] {
			continue
		}
		n++
		counted[c.ID] = true
		for _, i := range reverses.Of(c) { // de omgekeerde richting(en) horen bij hetzelfde woord
			counted[s.lib.Concepts[i].ID] = true
		}
	}
	return n
}

// Stats telt per stap en bepaalt hoeveel er nu klaarstaan en wanneer de volgende herhaling is
func (s *Session) Stats() Stats {
	now := time.Now()
	st := Stats{Mode: s.mode, Total: len(s.lib.Concepts), Intervals: engine.AnkiIntervals, NewPerDay: s.newPerDay}
	st.NewLeft = s.newAllowance(now)
	st.NewToday = s.introducedOn(engine.DayOf(now))
	for _, c := range s.lib.Concepts {
		if engine.IsNew(c) {
			// Nieuwe woorden staan in de wachtrij; ze tellen niet mee als stap of geplande herhaling
			st.NewQueue++
			continue
		}
		if c.StepIndex >= 0 && c.StepIndex < len(st.Steps) {
			st.Steps[c.StepIndex]++
		}
		if !c.NextReview.Before(now) && (st.NextDue == nil || c.NextReview.Before(*st.NextDue)) {
			t := c.NextReview
			st.NextDue = &t
		}
	}
	st.NewUnverified = s.unverifiedWords()
	// Te doen vandaag: precies wat de rondes per modus nog kunnen geven
	st.Due = len(engine.Candidates(s.lib.Concepts, now, engine.ModeReview, 0))
	st.DueNew = len(engine.Candidates(s.lib.Concepts, now, engine.ModeNew, st.NewLeft))
	for _, c := range engine.DueCandidates(s.lib.Concepts, now, engine.Unlimited) {
		if engine.IsNew(c) {
			st.NewAvailable++
		}
	}
	return st
}
