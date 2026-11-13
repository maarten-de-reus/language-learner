// De HTTP-server: de gedeelde staat, en hulpjes om JSON te lezen en te schrijven.

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"hungarian-language-learner/internal/catalog"
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

func statusFor(err error) int {
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, catalog.ErrDuplicate), errors.Is(err, catalog.ErrReverseAmbiguous), errors.Is(err, catalog.ErrContextTaken), errors.Is(err, catalog.ErrHintNeeded):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("ongeldige invoer"))
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
