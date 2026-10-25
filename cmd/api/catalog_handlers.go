package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"hungarian-language-learner/internal/catalog"
	"hungarian-language-learner/internal/storage"
)

// conceptRequest is de invoer voor check en toevoegen. Hungarian is een komma-gescheiden lijst.
type conceptRequest struct {
	Dutch            string            `json:"dutch"`
	Context          string            `json:"context"`
	Hungarian        string            `json:"hungarian"`
	Example          string            `json:"example"`
	ExtendForward    string            `json:"extendForward"` // bestaand NL->HU concept aanvullen; leeg = nieuwe betekenis
	ForwardHints     map[string]string `json:"forwardHints"`
	MergeReverseInto string            `json:"mergeReverseInto"`
	ReverseContext   *string           `json:"reverseContext"` // nil = zelfde hint als NL->HU
	ExistingHints    map[string]string `json:"existingHints"`
}

// checkResponse vertelt de frontend vóór het toevoegen of het een duplicaat is, of het
// Nederlandse woord al betekenissen heeft, en of het Hongaarse woord al betekenissen heeft
type checkResponse struct {
	Duplicate         bool            `json:"duplicate"`
	ForwardCandidates []catalog.Entry `json:"forwardCandidates"` // bestaande NL->HU betekenissen van het Nederlandse woord
	Reverse           string          `json:"reverse"`           // het Hongaarse woord dat de bron van HU->NL wordt
	Candidates        []catalog.Entry `json:"candidates"`        // bestaande HU->NL betekenissen daarvan
}

type entriesResponse struct {
	Entries []catalog.Entry `json:"entries"`
}

func (s *server) handleListConcepts(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, entriesResponse{s.entries()})
}

func (s *server) handleCheckConcept(w http.ResponseWriter, r *http.Request) {
	var req conceptRequest
	if !decode(w, r, &req) {
		return
	}
	nc, err := catalog.ParseInput(req.Dutch, req.Context, req.Hungarian)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	nc.ExtendForward = req.ExtendForward // bepaalt welk Hongaars woord de omgekeerde richting krijgt
	resp := checkResponse{
		Duplicate:         catalog.IsDuplicate(s.lib, nc),
		Reverse:           catalog.ReverseSourceFor(s.lib, nc),
		ForwardCandidates: []catalog.Entry{},
		Candidates:        []catalog.Entry{},
	}
	forward, reverse := map[string]bool{}, map[string]bool{}
	for _, i := range catalog.ForwardCandidates(s.lib, nc.Dutch) {
		forward[s.lib.Concepts[i].ID] = true
	}
	for _, i := range catalog.ReverseCandidates(s.lib, resp.Reverse) {
		reverse[s.lib.Concepts[i].ID] = true
	}
	for _, e := range catalog.Entries(s.lib) {
		if forward[e.ID] {
			resp.ForwardCandidates = append(resp.ForwardCandidates, e)
		}
		if reverse[e.ID] {
			resp.Candidates = append(resp.Candidates, e)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleAddConcept(w http.ResponseWriter, r *http.Request) {
	var req conceptRequest
	if !decode(w, r, &req) {
		return
	}
	nc, err := catalog.ParseInput(req.Dutch, req.Context, req.Hungarian)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	nc.Example = req.Example
	nc.ExtendForward = req.ExtendForward
	nc.ForwardHints = req.ForwardHints
	nc.MergeReverseInto = req.MergeReverseInto
	nc.ExistingHints = req.ExistingHints
	if req.ReverseContext != nil {
		nc.ReverseContext = *req.ReverseContext
	}

	s.mutate(w, http.StatusCreated, func() error {
		_, err := catalog.AddPair(s.lib, nc)
		return err
	})
}

func (s *server) handleUpdateConcept(w http.ResponseWriter, r *http.Request) {
	var edit catalog.Edit
	if !decode(w, r, &edit) {
		return
	}
	s.mutate(w, http.StatusOK, func() error {
		return catalog.Update(s.lib, r.PathValue("id"), edit)
	})
}

func (s *server) handleDeleteConcept(w http.ResponseWriter, r *http.Request) {
	var res catalog.DeleteResult
	withReverse := r.URL.Query().Get("reverse") == "1"
	s.mutate(w, http.StatusOK, func() (err error) {
		res, err = catalog.Delete(s.lib, r.PathValue("id"), withReverse)
		return err
	}, func() {
		// Pas na een geslaagde save: verwijderde concepten uit de lopende run halen
		// en de voortgang opschonen (voortgang.json bevat dan geen verwijderde concepten meer)
		s.session.Forget(res.Deleted...)
		if err := s.session.Save(); err != nil {
			log.Printf("Voortgang na verwijderen niet opgeslagen: %v", err)
		}
	})
}

// mutate voert een wijziging uit en slaat woorden.json op. Mislukt een van beide,
// dan wordt de woordenschat in het geheugen teruggezet, zodat geheugen en bestand gelijk blijven.
// onSaved draait alleen als de wijziging is opgeslagen.
func (s *server) mutate(w http.ResponseWriter, okStatus int, change func() error, onSaved ...func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.lib.Clone()
	if err := change(); err != nil {
		*s.lib = *snapshot
		writeError(w, statusFor(err), err)
		return
	}
	if err := storage.SaveWords(s.wordsPath, s.lib.Words, s.lib.Concepts); err != nil {
		*s.lib = *snapshot
		log.Printf("Fout bij opslaan woorden: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("woorden konden niet worden opgeslagen: "+err.Error()))
		return
	}
	for _, f := range onSaved {
		f()
	}
	writeJSON(w, okStatus, entriesResponse{s.entries()})
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
