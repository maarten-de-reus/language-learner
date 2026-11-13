// Nieuwe woorden: herhalen of nieuwe woorden leren, de daglimiet en extra woorden.

package quiz

import (
	"time"

	"hungarian-language-learner/internal/engine"
)

// SetMode kiest tussen herhalen en nieuwe woorden leren en begint een run in die modus. Een
// halve run gaat verloren, maar wat nog niet beantwoord is blijft gewoon aan de beurt.
// false bij een onbekende modus.
func (s *Session) SetMode(m engine.Mode) bool {
	if m != engine.ModeReview && m != engine.ModeNew {
		return false
	}
	if m != s.mode || s.Done() {
		s.mode = m
		s.startRun()
	}
	return true
}

// SetNewPerDay stelt de dagelijkse limiet voor nieuwe woorden in (0 = geen limiet).
// Is er in de huidige run nog niets beantwoord, dan wordt die opnieuw samengesteld.
func (s *Session) SetNewPerDay(n int) {
	s.newPerDay = n
	if s.pos == 0 && s.feedback == nil && len(s.results) == 0 {
		s.startRun()
	}
}

// AddExtraNew staat vandaag n extra nieuwe woorden toe en gaat naar nieuwe woorden leren.
// Is de run klaar, dan begint er meteen een nieuwe met die woorden.
func (s *Session) AddExtraNew(n int) {
	today := engine.DayOf(time.Now())
	if !s.extraDay.Equal(today) {
		s.extraNew, s.extraDay = 0, today
	}
	// Altijd n méér dan wat er nu al geïntroduceerd is, ook als je vandaag al over de limiet zit
	if used := s.introducedOn(today); s.newPerDay+s.extraNew < used {
		s.extraNew = used - s.newPerDay
	}
	s.extraNew += n
	if s.mode != engine.ModeNew || s.Done() {
		s.mode = engine.ModeNew
		s.startRun()
	}
}

// newAllowance is het aantal nieuwe woorden dat vandaag nog mag; engine.Unlimited zonder limiet
func (s *Session) newAllowance(now time.Time) int {
	if s.newPerDay == 0 {
		return engine.Unlimited
	}
	today := engine.DayOf(now)
	allowed := s.newPerDay
	if s.extraDay.Equal(today) {
		allowed += s.extraNew
	}
	return max(0, allowed-s.introducedOn(today))
}

// introducedOn telt de concepten die op deze herhaaldag voor het eerst gevraagd zijn
func (s *Session) introducedOn(day time.Time) int {
	n := 0
	for _, c := range s.lib.Concepts {
		if !engine.IsNew(c) && engine.DayOf(c.FirstReview.In(day.Location())).Equal(day) {
			n++
		}
	}
	return n
}
