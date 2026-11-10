package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"hungarian-language-learner/internal/catalog"
	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/quiz"
	"hungarian-language-learner/internal/storage"
)

// server is de HTTP-laag rond de woordenschat en één quiz-sessie;
// mu beschermt beide tegen gelijktijdige requests
type server struct {
	mu           sync.Mutex
	lib          *storage.Library
	wordsPath    string
	reviewsPath  string
	settingsPath string
	settings     storage.Settings
	session      *quiz.Session
	saveError    string
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
		Input   string `json:"input"`
		Unknown bool   `json:"unknown"` // "Weet ik niet"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "ongeldige invoer", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var fb *quiz.Feedback
	var err error
	if body.Unknown {
		fb, err = s.session.DontKnow()
	} else {
		fb, err = s.session.Answer(body.Input)
	}
	switch {
	case err != nil:
		s.saveError = "Voortgang of antwoordenlog kon niet worden opgeslagen: " + err.Error()
		log.Print(s.saveError)
	case fb != nil:
		s.saveError = ""
	}
	s.writeState(w)
}

// handlePracticeCheck kijkt een antwoord uit Kaarten na, zonder iets op te slaan
func (s *server) handlePracticeCheck(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID    string `json:"id"`
		Input string `json:"input"`
	}
	if !decode(w, r, &body) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	res, ok := s.session.Practice(body.ID, body.Input)
	if !ok {
		writeError(w, http.StatusNotFound, catalog.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleOverride is "Toch goed": het laatste foute antwoord telt alsnog als goed
func (s *server) handleOverride(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.session.Override(); err != nil {
		s.saveError = "Voortgang of antwoordenlog kon niet worden opgeslagen: " + err.Error()
		log.Print(s.saveError)
	}
	s.writeState(w)
}

// handleMode kiest tussen herhalen en nieuwe woorden leren
func (s *server) handleMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode engine.Mode `json:"mode"`
	}
	if !decode(w, r, &body) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.session.SetMode(body.Mode) {
		writeError(w, http.StatusBadRequest, errors.New("onbekende modus: kies review of new"))
		return
	}
	s.writeState(w)
}
