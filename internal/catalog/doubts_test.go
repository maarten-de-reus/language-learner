package catalog

import "testing"

func TestDoubts(t *testing.T) {
	lib := newLib()
	if _, err := add(t, lib, "kar", "", "szekér", func(nc *NewPair) { nc.WordType = "zn" }); err != nil {
		t.Fatal(err)
	}
	id := lib.Concepts[0].ID
	if err := SetDoubt(lib, id, "soep", "x"); err == nil {
		t.Error("onbekende sleutel moet een fout geven")
	}
	for _, key := range []string{"t", "k", "on"} {
		if err := SetDoubt(lib, id, key, "x"); err != nil {
			t.Fatal(err)
		}
	}
	if lib.Concepts[1].Doubts["t"] != "x" {
		t.Errorf("twijfel moet ook op de andere richting: %v", lib.Concepts[1].Doubts)
	}
	if err := ResolveDoubt(lib, id, "t", DoubtFix, "szekeret"); err != nil || lib.Concepts[1].Forms["t"] != "szekeret" {
		t.Errorf("verbeteren: %v %v", err, lib.Concepts[1].Forms)
	}
	if err := ResolveDoubt(lib, id, "k", DoubtSkip, ""); err != nil || lib.Concepts[0].Forms["k"] != "?" {
		t.Errorf("niet oefenen: %v %v", err, lib.Concepts[0].Forms)
	}
	if err := ResolveDoubt(lib, id, "on", DoubtCorrect, ""); err != nil || lib.Concepts[0].Doubts != nil || lib.Concepts[1].Forms["on"] != "" {
		t.Errorf("klopt toch: %v %v %v", err, lib.Concepts[0].Doubts, lib.Concepts[1].Forms)
	}
	if err := ResolveDoubt(lib, id, "t", DoubtCorrect, ""); err == nil {
		t.Error("geen twijfel meer: fout verwacht")
	}
	if err := ResolveDoubt(lib, id, "ban", DoubtFix, ""); err == nil {
		t.Error("zonder twijfel geen oplossing")
	}
}
