package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hungarian-language-learner/internal/grammar"
	"hungarian-language-learner/internal/storage"
)

// fewKnownWords: de testlijst heeft één woord; dat kent de leerling hier al een week (stap 2)
func fewKnownWords(t *testing.T, s *server) {
	old := grammar.MinWords
	grammar.MinWords = 1
	t.Cleanup(func() { grammar.MinWords = old })
	for i := range s.lib.Concepts {
		s.lib.Concepts[i].FirstReview, s.lib.Concepts[i].StepIndex = time.Now().AddDate(0, 0, -10), 2
	}
}

func TestGrammarFlow(t *testing.T) {
	s, h := newTestServer(t)
	fewKnownWords(t, s)
	s.grammarPath = filepath.Join(t.TempDir(), "regels.jsonl")

	// idő heeft nog geen woordsoort: geen opgave
	if code, _ := call(t, h, "GET", "/api/grammar/exercise", ""); code != http.StatusNotFound {
		t.Errorf("zonder woordsoort: verwacht 404, kreeg %d", code)
	}
	// Woordsoort zetten via Beheer (geldt ook voor de andere richting)
	if code, out := call(t, h, "PUT", "/api/concepts/c_nl_hu_weer", `{"answers":["idő"],"wordType":"zn"}`); code != http.StatusOK {
		t.Fatalf("woordsoort zetten: %d %s", code, out)
	}
	if s.lib.Concepts[1].WordType != "zn" {
		t.Errorf("de andere richting moet de woordsoort ook krijgen: %+v", s.lib.Concepts[1])
	}

	code, out := call(t, h, "GET", "/api/grammar", "")
	var view grammarView
	if err := json.Unmarshal(mustJSON(t, out), &view); code != http.StatusOK || err != nil || view.Typed != 1 || view.Untyped != 0 {
		t.Fatalf("overzicht: %d %v %+v", code, err, view)
	}

	// Alleen -ban/-ben aan: de opgave is idő + -ban/-ben
	if code, out := call(t, h, "PUT", "/api/grammar/active", `{"active":["ban"]}`); code != http.StatusOK {
		t.Fatalf("regels kiezen: %d %s", code, out)
	}
	if saved, _ := storage.LoadSettings(s.settingsPath); len(saved.GrammarRules) != 1 || saved.GrammarRules[0] != "ban" {
		t.Errorf("keuze niet opgeslagen: %+v", saved)
	}
	// Een andere instelling wijzigen laat de regels staan
	call(t, h, "PUT", "/api/settings", `{"newPerDay":5}`)
	if s.settings.GrammarRules == nil || s.settings.GrammarRules[0] != "ban" {
		t.Errorf("nieuwe woorden per dag wiste de regelkeuze: %+v", s.settings)
	}
	code, out = call(t, h, "GET", "/api/grammar/exercise", "")
	if code != http.StatusOK || string(out["rule"]) != `"ban"` || string(out["word"]) != `"idő"` {
		t.Fatalf("opgave: %d %s", code, out)
	}
	if strings.Contains(string(mustJSON(t, out)), "időben") {
		t.Error("de opgave mag het antwoord niet bevatten")
	}

	code, out = call(t, h, "POST", "/api/grammar/check", `{"rule":"ban","word":"idő","input":"időben"}`)
	if code != http.StatusOK || string(out["correct"]) != "true" {
		t.Fatalf("nakijken: %d %s", code, out)
	}
	if log, _ := storage.ReadGrammar(s.grammarPath); len(log) != 1 || !log[0].Correct || len(s.grammarLog) != 1 {
		t.Errorf("antwoord niet in regels.jsonl: %+v", log)
	}
	if code, _ := call(t, h, "PUT", "/api/grammar/active", `{"active":["soep"]}`); code != http.StatusBadRequest {
		t.Errorf("onbekende regel: verwacht 400, kreeg %d", code)
	}
	// Alles uit is iets anders dan de standaardkeuze
	call(t, h, "PUT", "/api/grammar/active", `{"active":[]}`)
	if code, _ := call(t, h, "GET", "/api/grammar/exercise", ""); code != http.StatusNotFound {
		t.Errorf("alles uit: verwacht geen opgave, kreeg %d", code)
	}
}

func mustJSON(t *testing.T, out map[string]json.RawMessage) []byte {
	t.Helper()
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGrammarDoubtFlow(t *testing.T) {
	s, h := newTestServer(t)
	fewKnownWords(t, s)
	s.grammarPath = filepath.Join(t.TempDir(), "regels.jsonl")
	call(t, h, "PUT", "/api/concepts/c_nl_hu_weer", `{"answers":["idő"],"wordType":"zn"}`)
	call(t, h, "PUT", "/api/grammar/active", `{"active":["t"]}`)
	if code, _ := call(t, h, "GET", "/api/grammar/exercise", ""); code != http.StatusOK {
		t.Fatalf("verwacht een opgave idő + -t, kreeg %d", code)
	}
	// Klopt niet: idő + -t wordt niet meer gevraagd
	if code, out := call(t, h, "POST", "/api/grammar/doubt", `{"rule":"t","word":"idő","person":2}`); code != http.StatusOK {
		t.Fatalf("melden: %d %s", code, out)
	}
	if got := s.lib.Concepts[1].Doubts["t"]; got != "időt" {
		t.Errorf("twijfel op de andere richting met de vorm van de regels: %q", got)
	}
	if code, _ := call(t, h, "GET", "/api/grammar/exercise", ""); code != http.StatusNotFound {
		t.Errorf("met twijfel geen opgave meer, kreeg %d", code)
	}
	// Controleren ziet de vormen, met de twijfel
	code, out := call(t, h, "POST", "/api/grammar/forms", `{"ids":["c_nl_hu_weer"]}`)
	if code != http.StatusOK || !strings.Contains(string(out["forms"]), `"doubt":"időt"`) {
		t.Fatalf("vormen: %d %s", code, out["forms"])
	}
	// Oplossen: klopt toch
	if code, out := call(t, h, "POST", "/api/concepts/c_nl_hu_weer/doubt", `{"key":"t","action":"correct"}`); code != http.StatusOK {
		t.Fatalf("oplossen: %d %s", code, out)
	}
	if code, _ := call(t, h, "GET", "/api/grammar/exercise", ""); code != http.StatusOK {
		t.Errorf("na oplossen weer een opgave, kreeg %d", code)
	}
	// Een woord uit een vaste lijst (Budapest bij -i) kan niet gemeld worden
	if code, _ := call(t, h, "POST", "/api/grammar/doubt", `{"rule":"i","word":"Budapest","person":2}`); code != http.StatusBadRequest {
		t.Errorf("vaste lijst: verwacht 400, kreeg %d", code)
	}
}

func TestGrammarLevel(t *testing.T) {
	s, h := newTestServer(t)
	call(t, h, "PUT", "/api/concepts/c_nl_hu_weer", `{"answers":["idő"],"wordType":"zn"}`)
	code, out := call(t, h, "PUT", "/api/grammar/level", `{"minStep":4}`)
	if code != http.StatusOK || string(out["minStep"]) != "4" {
		t.Fatalf("niveau: %d %s", code, out)
	}
	if saved, _ := storage.LoadSettings(s.settingsPath); saved.GrammarMinStep == nil || *saved.GrammarMinStep != 4 {
		t.Errorf("niveau niet opgeslagen: %+v", saved)
	}
	if code, _ := call(t, h, "PUT", "/api/grammar/level", `{"minStep":9}`); code != http.StatusBadRequest {
		t.Errorf("niveau 9: verwacht 400, kreeg %d", code)
	}
	// idő is nog nooit geoefend: -ban/-ben is niet klaar, met een uitleg
	call(t, h, "PUT", "/api/grammar/active", `{"active":["ban"]}`)
	code, out = call(t, h, "GET", "/api/grammar/exercise", "")
	if code != http.StatusNotFound || !strings.Contains(string(out["error"]), "nog niet") && !strings.Contains(string(out["error"]), "klaar") {
		t.Errorf("niet klaar: %d %s", code, out)
	}
}

func TestGrammarReviewFlow(t *testing.T) {
	s, h := newTestServer(t)
	fewKnownWords(t, s)
	s.grammarProgressPath = filepath.Join(t.TempDir(), "regels-voortgang.json")
	call(t, h, "PUT", "/api/concepts/c_nl_hu_weer", `{"answers":["idő"],"wordType":"zn"}`)
	call(t, h, "PUT", "/api/grammar/active", `{"active":["ban","t"]}`)

	code, out := call(t, h, "GET", "/api/state", "")
	if code != http.StatusOK || string(out["rulesDue"]) != "2" {
		t.Fatalf("twee nieuwe regels aan de beurt: %d %s", code, out["rulesDue"])
	}
	code, out = call(t, h, "GET", "/api/grammar/review", "")
	if code != http.StatusOK || !strings.Contains(string(out["due"]), `"id":"ban"`) || string(out["size"]) != "5" {
		t.Fatalf("herhaling: %d %s", code, out)
	}
	// -ban/-ben: 5 van 5 goed -> stap 0 (morgen), niet meer aan de beurt vandaag
	code, out = call(t, h, "POST", "/api/grammar/review/ban", `{"correct":5}`)
	if code != http.StatusOK || string(out["passed"]) != "true" || string(out["step"]) != "0" {
		t.Fatalf("klaar met ban: %d %s", code, out)
	}
	if saved, _ := storage.LoadGrammarProgress(s.grammarProgressPath); saved["ban"].FirstReview.IsZero() {
		t.Errorf("planning niet opgeslagen: %+v", saved)
	}
	if _, out := call(t, h, "GET", "/api/state", ""); string(out["rulesDue"]) != "1" {
		t.Errorf("nog één regel aan de beurt, kreeg %s", out["rulesDue"])
	}
	if code, _ := call(t, h, "POST", "/api/grammar/review/soep", `{"correct":5}`); code != http.StatusBadRequest {
		t.Errorf("onbekende regel: verwacht 400, kreeg %d", code)
	}
	if code, _ := call(t, h, "POST", "/api/grammar/review/t", `{"correct":9}`); code != http.StatusBadRequest {
		t.Errorf("meer goed dan opgaven: verwacht 400, kreeg %d", code)
	}
}
