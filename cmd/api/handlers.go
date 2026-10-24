package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"hungarian-language-learner/internal/quiz"
	"hungarian-language-learner/internal/storage"
)

// server is de HTTP-laag rond de woordenschat en één quiz-sessie;
// mu beschermt beide tegen gelijktijdige requests
type server struct {
	mu          sync.Mutex
	lib         *storage.Library
	wordsPath   string
	reviewsPath string
	session     *quiz.Session
	saveError   string
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

	fb, err := s.session.Answer(body.Input)
	switch {
	case err != nil:
		s.saveError = "Voortgang of antwoordenlog kon niet worden opgeslagen: " + err.Error()
		log.Print(s.saveError)
	case fb != nil:
		s.saveError = ""
	}
	s.writeState(w)
}
