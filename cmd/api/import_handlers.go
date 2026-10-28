package main

import (
	"errors"
	"log"
	"net/http"

	"hungarian-language-learner/internal/catalog"
	"hungarian-language-learner/internal/storage"
)

type importRequest struct {
	Text    string                       `json:"text"`
	Choices map[int]catalog.ImportChoice `json:"choices"` // per regelnummer
	Groups  []string                     `json:"groups"`  // voor alle woorden uit de import
}

type importResponse struct {
	Results    []catalog.ImportResult `json:"results"`
	OK         bool                   `json:"ok"` // geen fouten of open keuzes
	Added      int                    `json:"added"`
	Duplicates int                    `json:"duplicates"`
	Grouped    int                    `json:"grouped"` // waarvan in de groepen van de import gezet
	Errors     int                    `json:"errors"`
	Choices    int                    `json:"choices"`
	Entries    []catalog.Entry        `json:"entries,omitempty"` // na een geslaagde import
}

func summarize(results []catalog.ImportResult, ok bool) importResponse {
	resp := importResponse{Results: results, OK: ok}
	for _, r := range results {
		switch r.Status {
		case catalog.StatusAdded:
			resp.Added++
		case catalog.StatusDuplicate:
			resp.Duplicates++
			if r.Grouped {
				resp.Grouped++
			}
		case catalog.StatusError:
			resp.Errors++
		case catalog.StatusChoice:
			resp.Choices++
		}
	}
	return resp
}

// handleImportCheck doet de import op een kopie van de woordenschat: er verandert niets
func (s *server) handleImportCheck(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if !decode(w, r, &req) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := catalog.NormalizeGroups(s.lib, req.Groups); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	results, ok := catalog.Import(s.lib.Clone(), catalog.ParseImport(req.Text), req.Choices, req.Groups)
	writeJSON(w, http.StatusOK, summarize(results, ok))
}

// handleImport voegt alles in één keer toe, of niets: bij een fout of open keuze blijft alles zoals het was
func (s *server) handleImport(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if !decode(w, r, &req) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := catalog.NormalizeGroups(s.lib, req.Groups); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	snapshot := s.lib.Clone()
	results, ok := catalog.Import(s.lib, catalog.ParseImport(req.Text), req.Choices, req.Groups)
	resp := summarize(results, ok)
	if !ok {
		*s.lib = *snapshot
		writeJSON(w, http.StatusConflict, resp)
		return
	}
	if resp.Added == 0 && resp.Grouped == 0 {
		writeJSON(w, http.StatusOK, resp) // alleen duplicaten zonder nieuwe groep: niets op te slaan
		return
	}
	if err := storage.SaveWords(s.wordsPath, s.lib.Words, s.lib.Concepts); err != nil {
		*s.lib = *snapshot
		log.Printf("Import niet opgeslagen: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("woorden konden niet worden opgeslagen: "+err.Error()))
		return
	}
	resp.Entries = s.entries()
	writeJSON(w, http.StatusOK, resp)
}
