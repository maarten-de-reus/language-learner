package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func importBody(t *testing.T, text string, choices map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"text": text, "choices": choices})
	return string(b)
}

func TestImportCheckChangesNothing(t *testing.T) {
	s, h := newTestServer(t)
	code, out := call(t, h, "POST", "/api/import/check", importBody(t, "kávé;koffie\nidő;weer", nil))
	if code != http.StatusOK || string(out["added"]) != "1" || string(out["duplicates"]) != "1" || string(out["ok"]) != "true" {
		t.Fatalf("controleren: %d %s", code, out)
	}
	if len(s.lib.Concepts) != 2 {
		t.Errorf("controleren mag niets toevoegen, nu %d concepten", len(s.lib.Concepts))
	}
}

func TestImportAllOrNothing(t *testing.T) {
	s, h := newTestServer(t)
	// Regel 2 heeft een fout: er mag niets worden toegevoegd, ook regel 1 niet
	code, out := call(t, h, "POST", "/api/import", importBody(t, "kávé;koffie\nalleen", nil))
	if code != http.StatusConflict || string(out["errors"]) != "1" {
		t.Fatalf("verwacht 409 met 1 fout, kreeg %d %s", code, out)
	}
	if len(s.lib.Concepts) != 2 || savedConcepts(t, s) != 2 {
		t.Error("bij een fout mag er niets veranderen")
	}

	code, out = call(t, h, "POST", "/api/import", importBody(t, "kávé;koffie;drank;Kérek egy kávét.\ntea;thee", nil))
	if code != http.StatusOK || string(out["added"]) != "2" || out["entries"] == nil {
		t.Fatalf("verwacht 2 toegevoegd, kreeg %d %s", code, out)
	}
	if savedConcepts(t, s) != 6 {
		t.Errorf("woorden.json moet 6 concepten bevatten, kreeg %d", savedConcepts(t, s))
	}
}

func TestImportWithChoice(t *testing.T) {
	s, h := newTestServer(t)
	text := "idő;tijd;klok"
	// idő -> weer bestaat al zonder hint: er is een keuze nodig
	code, out := call(t, h, "POST", "/api/import", importBody(t, text, nil))
	if code != http.StatusConflict || string(out["choices"]) != "1" {
		t.Fatalf("verwacht een open keuze, kreeg %d %s", code, out)
	}
	choices := map[string]any{"1": map[string]any{"reverse": true, "existingHints": map[string]string{"c_hu_nl_idő": "buiten"}}}
	code, out = call(t, h, "POST", "/api/import", importBody(t, text, choices))
	if code != http.StatusOK || string(out["added"]) != "1" {
		t.Fatalf("met keuze: %d %s", code, out)
	}
	if c := s.lib.Concepts[s.lib.Index("c_hu_nl_idő")]; c.ContextSource != "buiten" {
		t.Errorf("bestaand idő moet hint 'buiten' krijgen, kreeg %q", c.ContextSource)
	}
}
