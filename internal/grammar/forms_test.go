package grammar

import "testing"

func TestAllForms(t *testing.T) {
	w := Word{Hungarian: "szekér", Type: "zn", Forms: map[string]string{"t": "szekeret"}, Doubts: map[string]string{"k": "szekérek"}}
	rows := AllForms(w)
	get := func(key string) FormRow {
		for _, r := range rows {
			if r.Key == key {
				return r
			}
		}
		t.Fatalf("geen rij %s", key)
		return FormRow{}
	}
	if r := get("t"); r.Text != "szekeret" || !r.Fixed {
		t.Errorf("afwijkende vorm: %+v", r)
	}
	if r := get("k"); r.Doubt != "szekérek" {
		t.Errorf("twijfel: %+v", r)
	}
	if r := get("ban"); r.Text != "szekérben" {
		t.Errorf("regel: %+v", r)
	}
	// Een vorm met twijfel wordt niet geoefend
	k, _ := RuleByID("k")
	if len(k.persons(w)) != 0 {
		t.Error("twijfel moet de vorm uit de oefening halen")
	}
	if verbs := AllForms(Word{Hungarian: "megy", Type: "ww"}); len(verbs) != 50 {
		t.Errorf("werkwoord: verwacht 6 per tijd (onbepaald, bepaald, verleden, verleden bepaald, fog, -hat) + -lak + gebiedende wijs (2×6) + infinitief, kreeg %d", len(verbs))
	}
}
