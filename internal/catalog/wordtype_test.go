package catalog

import "testing"

func TestWordTypeOnAddAndUpdate(t *testing.T) {
	lib := newLib()
	if _, err := add(t, lib, "huis", "", "ház", func(nc *NewPair) { nc.WordType = "zelfstandig naamwoord" }); err != nil {
		t.Fatal(err)
	}
	for _, c := range lib.Concepts {
		if c.WordType != "zn" {
			t.Errorf("%s: woordsoort %q, verwacht zn voor beide richtingen", c.ID, c.WordType)
		}
		lib.Concepts[lib.Index(c.ID)].Verified = true
	}
	if _, err := add(t, lib, "x", "", "y", func(nc *NewPair) { nc.WordType = "soep" }); err == nil {
		t.Error("onbekende woordsoort moet een fout geven")
	}

	id := lib.Concepts[0].ID
	bn := "bn"
	if err := Update(lib, id, Edit{Answers: TargetTexts(lib, lib.Concepts[0]), WordType: &bn}); err != nil {
		t.Fatal(err)
	}
	for _, c := range lib.Concepts {
		if c.WordType != "bn" || !c.Verified {
			t.Errorf("%s: woordsoort %q verified %v; verwacht bn voor beide richtingen, nog goedgekeurd", c.ID, c.WordType, c.Verified)
		}
	}
	// nil laat de woordsoort staan, "" wist hem
	if err := Update(lib, id, Edit{Answers: TargetTexts(lib, lib.Concepts[0])}); err != nil || lib.Concepts[1].WordType != "bn" {
		t.Errorf("nil moet ongewijzigd laten: %v %q", err, lib.Concepts[1].WordType)
	}
	empty := ""
	if err := Update(lib, id, Edit{Answers: TargetTexts(lib, lib.Concepts[0]), WordType: &empty}); err != nil || lib.Concepts[1].WordType != "" {
		t.Errorf("leeg moet wissen: %v %q", err, lib.Concepts[1].WordType)
	}
}
