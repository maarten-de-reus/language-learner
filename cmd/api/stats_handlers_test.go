package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"hungarian-language-learner/internal/stats"
)

func TestStatsAfterAnswering(t *testing.T) {
	s, h := newTestServer(t)
	id := s.session.Current().ID
	call(t, h, "POST", "/api/answer", `{"input":"fout"}`)

	code, out := call(t, h, "GET", "/api/stats", "")
	if code != http.StatusOK {
		t.Fatalf("verwacht 200, kreeg %d", code)
	}
	raw, _ := json.Marshal(out)
	var rep stats.Report
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Answers != 1 || rep.FirstAttempts != 1 || rep.FirstCorrect != 0 || !rep.PracticedToday || rep.Streak != 1 {
		t.Errorf("rapport na één fout antwoord: %+v", rep)
	}
	if len(rep.Hardest) != 1 || rep.Hardest[0].ID != id {
		t.Errorf("verwacht %s als moeilijkste woord, kreeg %+v", id, rep.Hardest)
	}
	if len(rep.Activity) != stats.ActivityDays || len(rep.Forecast) != stats.ForecastDays {
		t.Errorf("activiteit %d / vooruitblik %d dagen", len(rep.Activity), len(rep.Forecast))
	}

	// Het beheer toont dezelfde tellingen per concept
	_, out = call(t, h, "GET", "/api/concepts", "")
	for _, e := range entriesFrom(t, out["entries"]) {
		if e.ID == id && (e.Attempts != 1 || e.Wrong != 1) {
			t.Errorf("beheer: %s moet 1 poging en 1 fout tonen, kreeg %d / %d", id, e.Attempts, e.Wrong)
		}
	}
}

func TestStatsWithoutLog(t *testing.T) {
	_, h := newTestServer(t)
	code, out := call(t, h, "GET", "/api/stats", "")
	if code != http.StatusOK || string(out["answers"]) != "0" || string(out["streak"]) != "0" {
		t.Errorf("zonder antwoorden: verwacht een leeg rapport, kreeg %d %s", code, out)
	}
}
