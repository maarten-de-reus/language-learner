// De endpoints van de controle (goedkeuren, opmerking) en de voorrang uit Kaarten.

package main

import (
	"errors"
	"log"
	"net/http"

	"hungarian-language-learner/internal/catalog"
	"hungarian-language-learner/internal/storage"
)

// handleReview is de controle van een woord: goedkeuren, of niet met een opmerking. Het antwoord
// is klein (alleen de aangepaste concepten): de controleur klikt er veel achter elkaar.
func (s *server) handleReview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Verified bool   `json:"verified"`
		Note     string `json:"note"`
	}
	if !decode(w, r, &body) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.lib.Clone()
	ids, err := catalog.Review(s.lib, r.PathValue("id"), body.Verified, body.Note)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	if err := storage.SaveWords(s.wordsPath, s.lib.Words, s.lib.Concepts); err != nil {
		*s.lib = *snapshot
		log.Printf("Controle niet opgeslagen: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("woorden konden niet worden opgeslagen: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"ids": ids})
}

// handlePriority geeft een nieuw woord voorrang in de wachtrij (vanuit Kaarten). Dat is voortgang,
// dus het komt in voortgang.json.
func (s *server) handlePriority(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Priority bool `json:"priority"`
	}
	if !decode(w, r, &body) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	ids, err := catalog.SetPriority(s.lib, r.PathValue("id"), body.Priority)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	if err := s.session.Save(); err != nil {
		catalog.SetPriority(s.lib, r.PathValue("id"), !body.Priority)
		log.Printf("Voorrang niet opgeslagen: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("voortgang kon niet worden opgeslagen: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"ids": ids})
}
