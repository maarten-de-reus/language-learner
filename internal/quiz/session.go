// Package quiz bevat de oefensessie van de web-interface:
// een run van te herhalen concepten, antwoorden nakijken, voortgang bijwerken en opslaan.
package quiz

import (
	"errors"
	"slices"
	"strings"
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

// Feedback is de uitslag van het laatst gegeven antwoord
type Feedback struct {
	Correct    bool      `json:"correct"`
	AccentOnly bool      `json:"accentOnly"`
	Input      string    `json:"input"`
	Expected   []string  `json:"expected"`
	Step       int       `json:"step"`
	NextReview time.Time `json:"nextReview"`
	Retry      bool      `json:"retry"`     // dit was een oefenherhaling
	WillRetry  bool      `json:"willRetry"` // het concept komt aan het eind van de run nog een keer
	Example    string    `json:"example,omitempty"`
	// OtherMeaning is gevuld als het antwoord een goede vertaling is van een ándere betekenis
	// van hetzelfde woord (bijv. idő -> "tijd" terwijl "weer" bedoeld is). Het telt als fout.
	OtherMeaning *OtherMeaning `json:"otherMeaning,omitempty"`
}

// OtherMeaning beschrijft de andere betekenis waar het antwoord bij hoort
type OtherMeaning struct {
	Answer  string `json:"answer"`
	Context string `json:"context"`
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
	lib          *storage.Library // gedeeld met het beheer; na verwijderen roept het beheer Forget aan
	run          []runItem
	pos          int
	feedback     *Feedback // gevuld zodra het huidige concept beantwoord is
	results      []Result
	progressPath string
	runSize      int
	reviewLog    *storage.ReviewLog // optioneel: elk antwoord wordt hierin bijgehouden
}

// LogReviewsTo laat de sessie elk antwoord, ook oefenherhalingen, in het antwoordenlog schrijven
func (s *Session) LogReviewsTo(l *storage.ReviewLog) {
	s.reviewLog = l
}

// New maakt een sessie en start meteen de eerste run
func New(lib *storage.Library, progressPath string, runSize int) *Session {
	s := &Session{lib: lib, progressPath: progressPath, runSize: runSize}
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

// Answer kijkt het antwoord op het huidige concept na.
// Bij een eerste poging wordt de voortgang bijgewerkt en opgeslagen; bij een oefenherhaling niet.
// Een fout antwoord zet het concept (maximaal MaxRetries keer) achteraan de run.
// Geeft nil terug als er niets na te kijken valt (run klaar, of al beantwoord).
// Een fout betekent dat opslaan of het antwoordenlog mislukte; de voortgang is in het geheugen wel bijgewerkt.
func (s *Session) Answer(input string) (*Feedback, error) {
	if s.Done() || s.feedback != nil {
		return nil, nil
	}

	item := s.run[s.pos]
	concept := s.conceptOf(item)
	expected := s.targetTexts(concept)
	correct, accentOnly := engine.CheckAnswer(input, expected)
	stepBefore := concept.StepIndex

	var saveErr error
	if item.retry == 0 {
		engine.UpdateProgressSmart(concept, correct)
		saveErr = s.Save()
		item.result = len(s.results)
		s.results = append(s.results, Result{
			Prompt:   s.lib.Words[concept.SourceWordID].Text,
			Input:    input,
			Expected: expected,
			Correct:  correct,
		})
	} else {
		r := &s.results[item.result]
		r.Retries = item.retry
		r.RetryCorrect = correct
	}

	willRetry := !correct && item.retry < MaxRetries
	if willRetry {
		s.run = append(s.run, runItem{conceptID: item.conceptID, retry: item.retry + 1, result: item.result})
	}

	s.feedback = &Feedback{
		Correct:    correct,
		AccentOnly: accentOnly,
		Input:      input,
		Expected:   expected,
		Step:       concept.StepIndex,
		NextReview: concept.NextReview,
		Retry:      item.retry > 0,
		WillRetry:  willRetry,
		Example:    concept.Example,
	}
	if !correct {
		s.feedback.OtherMeaning = s.otherMeaning(concept, input)
	}
	return s.feedback, errors.Join(saveErr, s.logReview(concept, input, stepBefore))
}

// logReview schrijft het zojuist gegeven antwoord in het antwoordenlog (als dat ingesteld is)
func (s *Session) logReview(c *storage.TranslationConcept, input string, stepBefore int) error {
	if s.reviewLog == nil {
		return nil
	}
	fb := s.feedback
	return s.reviewLog.Append(storage.Review{
		Time:         time.Now(),
		ConceptID:    c.ID,
		Input:        input,
		Correct:      fb.Correct,
		AccentOnly:   fb.AccentOnly,
		OtherMeaning: fb.OtherMeaning != nil,
		Retry:        fb.Retry,
		StepBefore:   stepBefore,
		StepAfter:    c.StepIndex,
		NextReview:   c.NextReview,
	})
}

// otherMeaning zoekt een ander concept met hetzelfde bronwoord (taal + tekst, niet het ID)
// waarvoor de invoer wél het goede antwoord is
func (s *Session) otherMeaning(current *storage.TranslationConcept, input string) *OtherMeaning {
	src := s.lib.Words[current.SourceWordID]
	for _, c := range s.lib.Concepts {
		other := s.lib.Words[c.SourceWordID]
		if c.ID == current.ID || other.Language != src.Language || !strings.EqualFold(strings.TrimSpace(other.Text), strings.TrimSpace(src.Text)) {
			continue
		}
		targets := s.targetTexts(&c)
		if ok, _ := engine.CheckAnswer(input, targets); ok {
			return &OtherMeaning{Answer: strings.Join(targets, ", "), Context: c.ContextSource}
		}
	}
	return nil
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

// Stats telt per stap en bepaalt hoeveel er nu klaarstaan en wanneer de volgende herhaling is
func (s *Session) Stats() Stats {
	now := time.Now()
	st := Stats{Total: len(s.lib.Concepts), Intervals: engine.AnkiIntervals}
	for _, c := range s.lib.Concepts {
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
	chunk, _ := engine.GetNextChunk(s.lib.Concepts, s.runSize)
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
