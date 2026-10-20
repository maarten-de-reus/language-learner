package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

const (
	WordsPath    = "woorden.json"
	ProgressPath = "voortgang.json"
	RunSize      = 10
	Addr         = ":8080"
)

//go:embed web
var webFiles embed.FS

// server houdt de sessie van één gebruiker bij; mu beschermt alle velden
type server struct {
	mu        sync.Mutex
	words     map[string]storage.Word
	concepts  []storage.TranslationConcept
	idIndex   map[string]int // concept-ID -> index in concepts
	run       []int          // indices in concepts, zodat updates direct in de hoofdlijst landen
	pos       int
	feedback  *feedbackView // gevuld zodra het huidige concept beantwoord is
	results   []resultView
	saveError string
}

type cardView struct {
	ID       string `json:"id"`
	Prompt   string `json:"prompt"`
	Context  string `json:"context"`
	From     string `json:"from"`
	To       string `json:"to"`
	Step     int    `json:"step"`
	Position int    `json:"position"`
	RunSize  int    `json:"runSize"`
}

type feedbackView struct {
	Correct    bool      `json:"correct"`
	AccentOnly bool      `json:"accentOnly"`
	Input      string    `json:"input"`
	Expected   []string  `json:"expected"`
	Step       int       `json:"step"`
	NextReview time.Time `json:"nextReview"`
}

type resultView struct {
	Prompt   string   `json:"prompt"`
	Input    string   `json:"input"`
	Expected []string `json:"expected"`
	Correct  bool     `json:"correct"`
}

type statsView struct {
	Total     int                            `json:"total"`
	Due       int                            `json:"due"`
	Steps     [len(engine.AnkiIntervals)]int `json:"steps"`
	Intervals [len(engine.AnkiIntervals)]int `json:"intervals"`
	NextDue   *time.Time                     `json:"nextDue"`
}

type stateView struct {
	Card      *cardView     `json:"card"`
	Feedback  *feedbackView `json:"feedback"`
	Done      bool          `json:"done"`
	Results   []resultView  `json:"results"`
	Stats     statsView     `json:"stats"`
	SaveError string        `json:"saveError,omitempty"`
}

func main() {
	wordsPool, conceptsDB, err := storage.LoadData(WordsPath, ProgressPath)
	if err != nil {
		log.Fatalf("Fout bij laden data: %v", err)
	}

	s := &server{words: wordsPool, concepts: conceptsDB, idIndex: make(map[string]int)}
	for i, c := range conceptsDB {
		s.idIndex[c.ID] = i
	}
	s.startRun()

	static, _ := fs.Sub(webFiles, "web")
	http.Handle("/", http.FileServer(http.FS(static)))
	http.HandleFunc("GET /api/state", s.handle(func() {}))
	http.HandleFunc("POST /api/answer", s.handleAnswer)
	http.HandleFunc("POST /api/next", s.handle(s.next))
	http.HandleFunc("POST /api/run", s.handle(s.newRun))

	log.Printf("🌍 Web frontend gestart op http://localhost%s", Addr)
	log.Fatal(http.ListenAndServe(Addr, nil))
}

// handle voert een actie uit onder de lock en stuurt daarna de actuele staat terug
func (s *server) handle(action func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		action()
		s.writeState(w)
	}
}

func (s *server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "ongeldige invoer", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Geen actief concept of al beantwoord (bijv. dubbele submit): niets nakijken
	if s.pos >= len(s.run) || s.feedback != nil {
		s.writeState(w)
		return
	}

	concept := &s.concepts[s.run[s.pos]]
	expected := s.targetTexts(concept)
	correct, accentOnly := engine.CheckAnswer(body.Input, expected)

	engine.UpdateProgressSmart(concept, correct)
	if err := storage.SaveProgress(ProgressPath, s.concepts); err != nil {
		s.saveError = "Voortgang kon niet worden opgeslagen: " + err.Error()
		log.Print(s.saveError)
	} else {
		s.saveError = ""
	}

	s.feedback = &feedbackView{
		Correct:    correct,
		AccentOnly: accentOnly,
		Input:      body.Input,
		Expected:   expected,
		Step:       concept.StepIndex,
		NextReview: concept.NextReview,
	}
	s.results = append(s.results, resultView{
		Prompt:   s.words[concept.SourceWordID].Text,
		Input:    body.Input,
		Expected: expected,
		Correct:  correct,
	})
	s.writeState(w)
}

func (s *server) next() {
	if s.feedback == nil {
		return // eerst antwoorden
	}
	s.pos++
	s.feedback = nil
}

func (s *server) newRun() {
	if s.pos < len(s.run) {
		return // huidige run is nog bezig
	}
	s.startRun()
}

func (s *server) startRun() {
	chunk, _ := engine.GetNextChunk(s.concepts, RunSize)
	s.run = s.run[:0]
	for _, c := range chunk {
		s.run = append(s.run, s.idIndex[c.ID])
	}
	s.pos = 0
	s.feedback = nil
	s.results = nil
}

func (s *server) targetTexts(c *storage.TranslationConcept) []string {
	var texts []string
	for _, id := range c.TargetWordIDs {
		texts = append(texts, s.words[id].Text)
	}
	return texts
}

func (s *server) writeState(w http.ResponseWriter) {
	v := stateView{
		Feedback:  s.feedback,
		Results:   s.results,
		Stats:     s.stats(),
		SaveError: s.saveError,
	}
	if s.pos < len(s.run) {
		c := s.concepts[s.run[s.pos]]
		card := &cardView{
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
		v.Card = card
	} else {
		v.Done = true
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) stats() statsView {
	now := time.Now()
	st := statsView{Total: len(s.concepts), Intervals: engine.AnkiIntervals}
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
