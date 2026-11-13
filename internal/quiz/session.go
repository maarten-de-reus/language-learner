// Package quiz bevat de oefensessie van de web-interface:
// een run van te herhalen concepten, antwoorden nakijken, voortgang bijwerken en opslaan.
package quiz

import (
	"slices"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

// MaxRetries is hoe vaak een fout beantwoord concept aan het eind van de run opnieuw gevraagd wordt.
// Deze herhalingen zijn oefening: ze veranderen de planning niet, de echte toets is de volgende review.
const MaxRetries = 2

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
	Retry    bool   `json:"retry"`             // oefenherhaling binnen de run, telt niet mee voor de planning
	Example  string `json:"example,omitempty"` // alleen als de zin in de taal van de vraag is en dus het antwoord niet verraadt
}

// Result is één beantwoord concept binnen de huidige run; Correct gaat over de eerste poging
type Result struct {
	Prompt       string   `json:"prompt"`
	Input        string   `json:"input"`
	Expected     []string `json:"expected"`
	Correct      bool     `json:"correct"`
	Retries      int      `json:"retries"`      // aantal oefenherhalingen
	RetryCorrect bool     `json:"retryCorrect"` // bij een herhaling alsnog goed
}

// progress is de planning van een concept, om een antwoord terug te kunnen draaien
type progress struct {
	step int
	next time.Time
}

// runItem is één vraag in de run; retry > 0 betekent een oefenherhaling
type runItem struct {
	conceptID string // op ID, niet op positie: posities verschuiven als er een concept verwijderd wordt
	retry     int
	result    int // index in results (alleen voor herhalingen)
}

// conceptOf geeft het concept van een run-item in de gedeelde woordenschat, zodat
// wijzigingen (voortgang, bewerkingen in het beheer) direct in de hoofdlijst landen
func (s *Session) conceptOf(item runItem) *storage.TranslationConcept {
	if i := s.lib.Index(item.conceptID); i >= 0 {
		return &s.lib.Concepts[i]
	}
	return nil
}

// Forget haalt verwijderde concepten uit de lopende run. Was het huidige concept verwijderd,
// dan schuift de volgende vraag op zijn plek.
func (s *Session) Forget(ids ...string) {
	kept := make([]runItem, 0, len(s.run))
	pos := s.pos
	for i, item := range s.run {
		if !slices.Contains(ids, item.conceptID) {
			kept = append(kept, item)
			continue
		}
		if i < s.pos {
			pos--
		} else if i == s.pos {
			s.feedback = nil
		}
	}
	s.run, s.pos = kept, pos
}

// Session is niet veilig voor gelijktijdig gebruik; de aanroeper zorgt voor locking.
type Session struct {
	lib          *storage.Library // gedeeld met het beheer; na verwijderen roept het beheer Forget aan
	run          []runItem
	pos          int
	feedback     *Feedback // gevuld zodra het huidige concept beantwoord is
	undo         progress  // voortgang van het huidige concept vóór het antwoord, voor "Toch goed"
	results      []Result
	progressPath string
	runSize      int
	reviewLog    *storage.ReviewLog // optioneel: elk antwoord wordt hierin bijgehouden

	mode      engine.Mode // herhalen of nieuwe woorden leren; nooit door elkaar
	newPerDay int         // nieuwe woorden per dag; 0 = geen limiet
	extraNew  int         // extra nieuwe woorden vandaag ("Nog 10 nieuwe woorden")
	extraDay  time.Time   // herhaaldag waarop extraNew geldt
}

// LogReviewsTo laat de sessie elk antwoord, ook oefenherhalingen, in het antwoordenlog schrijven
func (s *Session) LogReviewsTo(l *storage.ReviewLog) {
	s.reviewLog = l
}

// New maakt een sessie en start meteen de eerste run
func New(lib *storage.Library, progressPath string, runSize int) *Session {
	s := &Session{lib: lib, progressPath: progressPath, runSize: runSize, newPerDay: storage.DefaultNewPerDay, mode: engine.ModeReview}
	// Eerst herhalen; is er niets te herhalen, dan meteen nieuwe woorden
	if len(engine.Candidates(lib.Concepts, time.Now(), engine.ModeReview, 0)) == 0 {
		s.mode = engine.ModeNew
	}
	s.startRun()
	return s
}

// load telt per herhaaldag hoeveel concepten er gepland staan, voor de spreiding van herhalingen
func (s *Session) load(except string) engine.Load {
	counts := map[time.Time]int{}
	for _, c := range s.lib.Concepts {
		if c.ID != except && !engine.IsNew(c) {
			counts[engine.DayOf(c.NextReview.In(time.Local))]++
		}
	}
	return func(day time.Time) int { return counts[day] }
}

// Done is true als de run klaar is (of leeg was omdat er niets te herhalen viel)
func (s *Session) Done() bool { return s.pos >= len(s.run) }

// Results geeft de antwoorden van de huidige run
func (s *Session) Results() []Result { return s.results }

// Current geeft het concept dat nu gevraagd wordt, of nil als de run klaar is
func (s *Session) Current() *Card {
	if s.Done() {
		return nil
	}
	item := s.run[s.pos]
	c := s.conceptOf(item)
	card := &Card{
		ID:       c.ID,
		Prompt:   s.lib.Words[c.SourceWordID].Text,
		Context:  c.ContextSource,
		From:     s.lib.Words[c.SourceWordID].Language,
		Step:     c.StepIndex,
		Position: s.pos + 1,
		RunSize:  len(s.run),
		Retry:    item.retry > 0,
	}
	if c.Example != "" && c.ExampleLang == card.From {
		card.Example = c.Example
	}
	if len(c.TargetWordIDs) > 0 {
		card.To = s.lib.Words[c.TargetWordIDs[0]].Language
	}
	return card
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
	return storage.SaveProgress(s.progressPath, s.lib.Concepts)
}

func (s *Session) startRun() {
	chunk := engine.NextChunk(s.lib.Concepts, s.runSize, s.mode, s.newAllowance(time.Now()))
	s.run = s.run[:0]
	for _, c := range chunk {
		s.run = append(s.run, runItem{conceptID: c.ID})
	}
	s.pos = 0
	s.feedback = nil
	s.results = nil
}

func (s *Session) targetTexts(c *storage.TranslationConcept) []string {
	var texts []string
	for _, id := range c.TargetWordIDs {
		texts = append(texts, s.lib.Words[id].Text)
	}
	return texts
}
