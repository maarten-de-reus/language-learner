package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"hungarian-language-learner/internal/storage"
)

func TestGrammarFlow(t *testing.T) {
	s, h := newTestServer(t)
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
