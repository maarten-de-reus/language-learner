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

func TestGroupsViaAPI(t *testing.T) {
	_, h := newTestServer(t)
	code, _ := call(t, h, "POST", "/api/concepts", `{"dutch":"thee","hungarian":"tea","groups":["Drinken","Week 1"]}`)
	if code != http.StatusCreated {
		t.Fatalf("toevoegen met groepen: %d", code)
	}
	b, _ := json.Marshal(map[string]any{"text": "kávé;koffie", "groups": []string{"drinken"}})
	if code, out := call(t, h, "POST", "/api/import", string(b)); code != http.StatusOK {
		t.Fatalf("import met groep: %d %s", code, out)
	}
	_, out := call(t, h, "GET", "/api/groups", "")
	var groups []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	json.Unmarshal(out["groups"], &groups)
	if len(groups) != 2 || groups[0].Name != "Drinken" || groups[0].Count != 4 || groups[1].Name != "Week 1" {
		t.Errorf("groepen: %+v", groups)
	}

	// Bewerken: groep eraf, ook in de andere richting
	code, out = call(t, h, "PUT", "/api/concepts/c_nl_hu_thee", `{"context":"","answers":["tea"],"groups":["Drinken"]}`)
	if code != http.StatusOK {
		t.Fatalf("bewerken: %d %s", code, out)
	}
	_, out = call(t, h, "GET", "/api/groups", "")
	json.Unmarshal(out["groups"], &groups)
	if len(groups) != 1 {
		t.Errorf("Week 1 moet weg zijn (beide richtingen), kreeg %+v", groups)
	}

	if code, _ := call(t, h, "POST", "/api/concepts", `{"dutch":"bier","hungarian":"sör","groups":["a, b"]}`); code != http.StatusBadRequest {
		t.Errorf("ongeldige groepsnaam: verwacht 400, kreeg %d", code)
	}
}
