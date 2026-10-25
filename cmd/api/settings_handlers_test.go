package main

import (
	"net/http"
	"testing"

	"hungarian-language-learner/internal/storage"
)

func TestSettingsAreSavedAndApplied(t *testing.T) {
	s, h := newTestServer(t)

	code, out := call(t, h, "GET", "/api/settings", "")
	if code != http.StatusOK || string(out["newPerDay"]) != "20" {
		t.Fatalf("standaard: verwacht 20, kreeg %d %s", code, out)
	}

	code, out = call(t, h, "PUT", "/api/settings", `{"newPerDay":0}`)
	if code != http.StatusOK || string(out["newPerDay"]) != "0" {
		t.Fatalf("0 (geen limiet) moet kunnen, kreeg %d %s", code, out)
	}
	if saved, _ := storage.LoadSettings(s.settingsPath); saved.NewPerDay != 0 {
		t.Errorf("instellingen.json niet bijgewerkt: %+v", saved)
	}
	if st := s.session.Stats(); st.NewPerDay != 0 || st.NewLeft != -1 {
		t.Errorf("de sessie moet de nieuwe limiet gebruiken: %+v", st)
	}

	if code, _ := call(t, h, "PUT", "/api/settings", `{"newPerDay":-5}`); code != http.StatusBadRequest {
		t.Errorf("negatieve limiet: verwacht 400, kreeg %d", code)
	}
}

func TestExtraNewWords(t *testing.T) {
	s, h := newTestServer(t)
	call(t, h, "PUT", "/api/settings", `{"newPerDay":1}`)
	before := s.session.Stats().NewLeft

	code, out := call(t, h, "POST", "/api/new/extra", `{}`)
	if code != http.StatusOK || out["stats"] == nil {
		t.Fatalf("verwacht de quizstaat terug, kreeg %d %s", code, out)
	}
	if got := s.session.Stats().NewLeft; got != before+ExtraNewWords {
		t.Errorf("verwacht %d extra toegestaan, kreeg %d -> %d", ExtraNewWords, before, got)
	}
}
