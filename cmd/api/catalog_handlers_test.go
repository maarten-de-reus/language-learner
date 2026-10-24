package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hungarian-language-learner/internal/catalog"
	"hungarian-language-learner/internal/quiz"
	"hungarian-language-learner/internal/storage"
)

const testWords = `{
  "words": [
    {"id": "w_nl_weer", "language": "nl", "text": "weer"},
    {"id": "w_hu_idő", "language": "hu", "text": "idő"}
  ],
  "concepts": [
    {"id": "c_nl_hu_weer", "source_word_id": "w_nl_weer", "context_source": "", "target_word_ids": ["w_hu_idő"]},
    {"id": "c_hu_nl_idő", "source_word_id": "w_hu_idő", "context_source": "", "target_word_ids": ["w_nl_weer"]}
  ]
}`

func newTestServer(t *testing.T) (*server, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	wordsPath := filepath.Join(dir, "woorden.json")
	progressPath := filepath.Join(dir, "voortgang.json")
	if err := os.WriteFile(wordsPath, []byte(testWords), 0644); err != nil {
		t.Fatal(err)
	}
	lib, err := storage.LoadLibrary(wordsPath, progressPath)
	if err != nil {
		t.Fatal(err)
	}
	session := quiz.New(lib, progressPath, 10)
	reviewsPath := filepath.Join(dir, "antwoorden.jsonl")
	session.LogReviewsTo(storage.NewReviewLog(reviewsPath))
	s := &server{lib: lib, wordsPath: wordsPath, reviewsPath: reviewsPath, session: session}
	return s, s.routes()
}

func call(t *testing.T, h http.Handler, method, path, body string) (int, map[string]json.RawMessage) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
	var out map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: geen JSON: %s", method, path, rr.Body.String())
	}
	return rr.Code, out
}

func entriesFrom(t *testing.T, raw json.RawMessage) []catalog.Entry {
	t.Helper()
	var entries []catalog.Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}

func savedConcepts(t *testing.T, s *server) int {
	t.Helper()
	_, concepts, err := storage.LoadData(s.wordsPath, filepath.Join(t.TempDir(), "leeg.json"))
	if err != nil {
		t.Fatal(err)
	}
	return len(concepts)
}

func TestListConcepts(t *testing.T) {
	_, h := newTestServer(t)
	code, out := call(t, h, "GET", "/api/concepts", "")
	if code != http.StatusOK || len(entriesFrom(t, out["entries"])) != 2 {
		t.Fatalf("verwacht 200 met 2 entries, kreeg %d %s", code, out)
	}
}

func TestCheckFindsDuplicateAndOtherMeanings(t *testing.T) {
	_, h := newTestServer(t)

	_, out := call(t, h, "POST", "/api/concepts/check", `{"dutch":"Weer","context":"","hungarian":"idő"}`)
	if string(out["duplicate"]) != "true" {
		t.Errorf("verwacht duplicaat, kreeg %s", out["duplicate"])
	}

	_, out = call(t, h, "POST", "/api/concepts/check", `{"dutch":"tijd","context":"klok","hungarian":"idő"}`)
	cands := entriesFrom(t, out["candidates"])
	if string(out["duplicate"]) != "false" || len(cands) != 1 || cands[0].ID != "c_hu_nl_idő" {
		t.Errorf("verwacht één bestaande betekenis van idő, kreeg %s", out)
	}

	code, _ := call(t, h, "POST", "/api/concepts/check", `{"dutch":"tijd","hungarian":" , "}`)
	if code != http.StatusBadRequest {
		t.Errorf("lege vertaling: verwacht 400, kreeg %d", code)
	}
}

func TestAddConceptSavesAndIsQuizzable(t *testing.T) {
	s, h := newTestServer(t)
	body := `{"dutch":"tijd","context":"klok","hungarian":"idő","example":"Nincs időm.",
	          "existingHints":{"c_hu_nl_idő":"buiten"}}`
	code, out := call(t, h, "POST", "/api/concepts", body)
	if code != http.StatusCreated {
		t.Fatalf("verwacht 201, kreeg %d %s", code, out)
	}
	if n := len(entriesFrom(t, out["entries"])); n != 4 {
		t.Errorf("verwacht 4 entries, kreeg %d", n)
	}
	if n := savedConcepts(t, s); n != 4 {
		t.Errorf("verwacht 4 concepten in woorden.json, kreeg %d", n)
	}
	if c := s.lib.Concepts[s.lib.Index("c_hu_nl_idő")]; c.ContextSource != "buiten" {
		t.Errorf("bestaande betekenis moet hint 'buiten' krijgen, kreeg %q", c.ContextSource)
	}

	// Nieuw concept zit in dezelfde woordenschat als de quiz: na deze run komt het aan de beurt
	for !s.session.Done() {
		s.session.Answer("x")
		s.session.Next()
	}
	s.session.NewRun()
	found := false
	for !s.session.Done() {
		// Beide richtingen zijn nieuw; welke in deze run zit is willekeurig (de andere volgt later)
		if id := s.session.Current().ID; strings.HasPrefix(id, "c_nl_hu_tijd") || strings.HasPrefix(id, "c_hu_nl_idő_klok") {
			found = true
		}
		s.session.Answer("x")
		s.session.Next()
	}
	if !found {
		t.Error("nieuw concept moet meteen geoefend kunnen worden")
	}
}

func TestAddDuplicateIsConflictAndNotSaved(t *testing.T) {
	s, h := newTestServer(t)
	code, out := call(t, h, "POST", "/api/concepts", `{"dutch":"weer","context":"","hungarian":"idő"}`)
	if code != http.StatusConflict || out["error"] == nil {
		t.Fatalf("verwacht 409 met foutmelding, kreeg %d %s", code, out)
	}
	if len(s.lib.Concepts) != 2 || savedConcepts(t, s) != 2 {
		t.Error("bij een duplicaat mag er niets veranderen")
	}
}

func TestUpdateConceptWithAccentedID(t *testing.T) {
	s, h := newTestServer(t)
	path := "/api/concepts/" + url.PathEscape("c_hu_nl_idő")
	code, out := call(t, h, "PUT", path, `{"context":"buiten ☀️","answers":["weer"],"example":"Milyen az idő?"}`)
	if code != http.StatusOK {
		t.Fatalf("verwacht 200, kreeg %d %s", code, out)
	}
	if c := s.lib.Concepts[s.lib.Index("c_hu_nl_idő")]; c.ContextSource != "buiten ☀️" || c.Example != "Milyen az idő?" {
		t.Errorf("concept niet aangepast: %+v", c)
	}

	code, _ = call(t, h, "PUT", "/api/concepts/bestaat-niet", `{"answers":["x"]}`)
	if code != http.StatusNotFound {
		t.Errorf("onbekend ID: verwacht 404, kreeg %d", code)
	}
}

func TestFailedSaveRollsBack(t *testing.T) {
	s, h := newTestServer(t)
	s.wordsPath = filepath.Join(t.TempDir(), "bestaat-niet", "woorden.json")

	code, _ := call(t, h, "POST", "/api/concepts", `{"dutch":"tijd","context":"klok","hungarian":"idő"}`)
	if code != http.StatusInternalServerError {
		t.Fatalf("verwacht 500, kreeg %d", code)
	}
	if len(s.lib.Concepts) != 2 || len(s.lib.Words) != 2 {
		t.Errorf("geheugen moet teruggezet zijn, kreeg %d concepten / %d woorden", len(s.lib.Concepts), len(s.lib.Words))
	}
}

func TestDeleteConceptWithReverse(t *testing.T) {
	s, h := newTestServer(t)
	current := s.session.Current().ID

	code, out := call(t, h, "DELETE", "/api/concepts/"+url.PathEscape("c_nl_hu_weer")+"?reverse=1", "{}")
	if code != http.StatusOK {
		t.Fatalf("verwacht 200, kreeg %d %s", code, out)
	}
	if n := len(entriesFrom(t, out["entries"])); n != 0 {
		t.Errorf("beide richtingen moeten weg zijn, nog %d over", n)
	}
	if savedConcepts(t, s) != 0 {
		t.Error("woorden.json moet bijgewerkt zijn")
	}
	if !s.session.Done() {
		t.Errorf("het verwijderde concept %s mag niet meer gevraagd worden", current)
	}

	code, _ = call(t, h, "DELETE", "/api/concepts/c_nl_hu_weer", "{}")
	if code != http.StatusNotFound {
		t.Errorf("nogmaals verwijderen: verwacht 404, kreeg %d", code)
	}
}

func TestDeleteSingleDirectionKeepsReverseQuizzable(t *testing.T) {
	s, h := newTestServer(t)
	code, out := call(t, h, "DELETE", "/api/concepts/c_nl_hu_weer", "{}")
	if code != http.StatusOK {
		t.Fatalf("verwacht 200, kreeg %d %s", code, out)
	}
	entries := entriesFrom(t, out["entries"])
	if len(entries) != 1 || entries[0].ID != "c_hu_nl_idő" || len(entries[0].Reverse) != 0 {
		t.Errorf("alleen idő -> weer moet overblijven, kreeg %+v", entries)
	}
	if c := s.session.Current(); c != nil && c.ID == "c_nl_hu_weer" {
		t.Error("het verwijderde concept mag niet meer gevraagd worden")
	}
}
