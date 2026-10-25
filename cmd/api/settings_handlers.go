package main

import (
	"net/http"

	"hungarian-language-learner/internal/storage"
)

type settingsView struct {
	NewPerDay int `json:"newPerDay"` // 0 = geen limiet
}

func (s *server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, settingsView{NewPerDay: s.settings.NewPerDay})
}

func (s *server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var v settingsView
	if !decode(w, r, &v) {
		return
	}
	next := storage.Settings{NewPerDay: v.NewPerDay}
	if err := next.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settingsPath != "" {
		if err := storage.SaveSettings(s.settingsPath, next); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	s.settings = next
	s.session.SetNewPerDay(next.NewPerDay)
	writeJSON(w, http.StatusOK, settingsView{NewPerDay: next.NewPerDay})
}
