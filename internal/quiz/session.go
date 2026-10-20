// Package quiz bevat de oefensessie die door alle frontends (CLI, web) gedeeld wordt:
// een run van te herhalen concepten, antwoorden nakijken, voortgang bijwerken en opslaan.
package quiz

import (
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

// Card is het concept dat nu gevraagd wordt
type Card struct {
	ID       string `json:"id"`
	Prompt   string `json:"prompt"`
	Context  string `json:"context"`
	From     string `json:"from"`
	To       string `json:"to"`
	Step     int    `json:"step"`
	Position int    `json:"position"`
	RunSize  int    `json:"runSize"`
}

// Feedback is de uitslag van het laatst gegeven antwoord
type Feedback struct {
	Correct    bool      `json:"correct"`
	AccentOnly bool      `json:"accentOnly"`
	Input      string    `json:"input"`
	Expected   []string  `json:"expected"`
	Step       int       `json:"step"`
	NextReview time.Time `json:"nextReview"`
}

// Result is één beantwoord concept binnen de huidige run
type Result struct {
	Prompt   string   `json:"prompt"`
	Input    string   `json:"input"`
	Expected []string `json:"expected"`
	Correct  bool     `json:"correct"`
}

// Stats geeft een overzicht van alle concepten
type Stats struct {
	Total     int                            `json:"total"`
	Due       int                            `json:"due"`
	Steps     [len(engine.AnkiIntervals)]int `json:"steps"`
	Intervals [len(engine.AnkiIntervals)]int `json:"intervals"`
	NextDue   *time.Time                     `json:"nextDue"`
}

// Session is niet veilig voor gelijktijdig gebruik; de aanroeper zorgt voor locking.
type Session struct {
	words        map[string]storage.Word
	concepts     []storage.TranslationConcept
	idIndex      map[string]int // concept-ID -> index in concepts
	run          []int          // indices in concepts, zodat updates direct in de hoofdlijst landen
	pos          int
	feedback     *Feedback // gevuld zodra het huidige concept beantwoord is
	results      []Result
	progressPath string
	runSize      int
}

// New maakt een sessie en start meteen de eerste run
func New(words map[string]storage.Word, concepts []storage.TranslationConcept, progressPath string, runSize int) *Session {
	s := &Session{
		words:        words,
		concepts:     concepts,
		idIndex:      make(map[string]int, len(concepts)),
		progressPath: progressPath,
		runSize:      runSize,
	}
	for i, c := range concepts {
		s.idIndex[c.ID] = i
	}
	s.startRun()
	return s
}

// Done is true als de run klaar is (of leeg was omdat er niets te herhalen viel)
func (s *Session) Done() bool { return s.pos >= len(s.run) }

// Feedback geeft de uitslag van het huidige concept, of nil als het nog niet beantwoord is
func (s *Session) Feedback() *Feedback { return s.feedback }

// Results geeft de antwoorden van de huidige run
func (s *Session) Results() []Result { return s.results }

// Current geeft het concept dat nu gevraagd wordt, of nil als de run klaar is
func (s *Session) Current() *Card {
	if s.Done() {
		return nil
	}
	c := s.concepts[s.run[s.pos]]
	card := &Card{
		ID:       c.ID,
		Prompt:   s.words[c.SourceWordID].Text,
		Context:  c.ContextSource,
		From:     s.words[c.SourceWordID].Language,
		Step:     c.StepIndex,
		Position: s.pos + 1,
		RunSize:  len(s.run),
	}
	if len(c.TargetWordIDs) > 0 {
		card.To = s.words[c.TargetWordIDs[0]].Language
	}
	return card
}

// Answer kijkt het antwoord op het huidige concept na, werkt de voortgang bij en slaat die op.
// Geeft nil terug als er niets na te kijken valt (run klaar, of al beantwoord).
// Een fout betekent dat opslaan mislukte; de voortgang is in het geheugen wel bijgewerkt.
func (s *Session) Answer(input string) (*Feedback, error) {
	if s.Done() || s.feedback != nil {
		return nil, nil
	}

	concept := &s.concepts[s.run[s.pos]]
	expected := s.targetTexts(concept)
	correct, accentOnly := engine.CheckAnswer(input, expected)
	engine.UpdateProgressSmart(concept, correct)

	s.feedback = &Feedback{
		Correct:    correct,
		AccentOnly: accentOnly,
		Input:      input,
		Expected:   expected,
		Step:       concept.StepIndex,
		NextReview: concept.NextReview,
	}
	s.results = append(s.results, Result{
		Prompt:   s.words[concept.SourceWordID].Text,
		Input:    input,
		Expected: expected,
		Correct:  correct,
	})
	return s.feedback, s.Save()
}

// Next gaat naar het volgende concept; kan alleen nadat het huidige beantwoord is
func (s *Session) Next() bool {
	if s.feedback == nil {
		return false
	}
	s.pos++
	s.feedback = nil
	return true
}

// NewRun start een nieuwe run; kan alleen als de huidige klaar is
func (s *Session) NewRun() bool {
	if !s.Done() {
		return false
	}
	s.startRun()
	return true
}

// Save schrijft de voortgang van alle concepten weg
func (s *Session) Save() error {
	return storage.SaveProgress(s.progressPath, s.concepts)
}

// Stats telt per stap en bepaalt hoeveel er nu klaarstaan en wanneer de volgende herhaling is
func (s *Session) Stats() Stats {
	now := time.Now()
	st := Stats{Total: len(s.concepts), Intervals: engine.AnkiIntervals}
	for _, c := range s.concepts {
		if c.StepIndex >= 0 && c.StepIndex < len(st.Steps) {
			st.Steps[c.StepIndex]++
		}
		if c.NextReview.Before(now) {
			st.Due++
		} else if st.NextDue == nil || c.NextReview.Before(*st.NextDue) {
			t := c.NextReview
			st.NextDue = &t
		}
	}
	return st
}

func (s *Session) startRun() {
	chunk, _ := engine.GetNextChunk(s.concepts, s.runSize)
	s.run = s.run[:0]
	for _, c := range chunk {
		s.run = append(s.run, s.idIndex[c.ID])
	}
	s.pos = 0
	s.feedback = nil
	s.results = nil
}

func (s *Session) targetTexts(c *storage.TranslationConcept) []string {
	var texts []string
	for _, id := range c.TargetWordIDs {
		texts = append(texts, s.words[id].Text)
	}
	return texts
}
