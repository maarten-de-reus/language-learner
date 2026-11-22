package grammar

import (
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

func TestRuleReviews(t *testing.T) {
	fewWords(t)
	words := []Word{{Hungarian: "ház", Type: "zn", Step: 3}, {Hungarian: "olvas", Type: "ww", Step: 3}}
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.Local)
	progress := map[string]storage.RuleProgress{
		"ba":  {Step: 2, NextReview: now.Add(-48 * time.Hour)}, // te laat
		"bol": {Step: 2, NextReview: now.Add(72 * time.Hour)},  // nog niet
		"ban": {Step: 1, NextReview: now.Add(-1 * time.Hour)},  // vandaag
	}
	due, done := DueRules(words, []string{"ban", "ba", "bol", "ww-onbep", "kor"}, progress, now)
	ids := []string{}
	for _, d := range due {
		ids = append(ids, d.ID)
	}
	if done != 0 || len(ids) != 4 || ids[0] != "ba" || ids[1] != "ban" || !due[2].New || !due[3].New {
		t.Fatalf("aan de beurt: %v (%+v), vandaag gedaan %d", ids, due, done)
	}
	// Limiet per dag
	for i := 0; i < RulesPerDay; i++ {
		progress["x"+string(rune('a'+i))] = storage.RuleProgress{LastReview: now.Add(-time.Minute), NextReview: now.Add(72 * time.Hour)}
	}
	if due, done := DueRules(words, []string{"ban", "ba"}, progress, now); len(due) != 0 || done != RulesPerDay {
		t.Errorf("na %d regels vandaag niets meer: %+v %d", RulesPerDay, due, done)
	}

	// Herhaling: 4 van 5 goed = een stap verder, 3 = twee terug en morgen
	p, passed := ApplyReview(storage.RuleProgress{}, 5, now)
	if !passed || p.Step != 0 || p.FirstReview.IsZero() {
		t.Errorf("eerste keer goed: %+v", p)
	}
	p, _ = ApplyReview(storage.RuleProgress{Step: 3, FirstReview: now}, 4, now)
	if p.Step != 4 || p.NextReview.Before(now.AddDate(0, 0, 39)) {
		t.Errorf("stap 3 -> 4 (40 dagen): %+v", p)
	}
	p, passed = ApplyReview(storage.RuleProgress{Step: 3, FirstReview: now}, 3, now)
	if passed || p.Step != 1 || p.NextReview.After(now.AddDate(0, 0, 2)) {
		t.Errorf("fout: twee stappen terug, morgen weer: %+v", p)
	}
	p, _ = ApplyReview(storage.RuleProgress{Step: maxRuleStep, FirstReview: now}, 5, now)
	if p.Step != maxRuleStep {
		t.Errorf("regels gaan niet verder dan 90 dagen: %+v", p)
	}
}
