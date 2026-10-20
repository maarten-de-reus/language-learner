package main

import (
	"errors"
	"slices"
	"testing"

	"hungarian-language-learner/internal/storage"
)

func TestAddConceptPairCreatesBothDirections(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, res, err := addConceptPair(words, nil, "bank", "Meubel", "pad, kanapé")
	if err != nil {
		t.Fatal(err)
	}
	if len(concepts) != 2 {
		t.Fatalf("verwacht 2 concepten, kreeg %d", len(concepts))
	}
	if got := concepts[0].TargetWordIDs; !slices.Equal(got, []string{"w_hu_pad", "w_hu_kanapé"}) {
		t.Errorf("NL->HU targets: %v", got)
	}
	if concepts[1].ID != "c_hu_nl_pad_meubel" || concepts[1].SourceWordID != "w_hu_pad" {
		t.Errorf("HU->NL concept: %+v", concepts[1])
	}
	if res.ReverseMerged {
		t.Error("ReverseMerged zou false moeten zijn")
	}
}

func TestAddConceptPairRejectsEmptyTranslation(t *testing.T) {
	for _, input := range []string{"", ",", " , ,"} {
		words := map[string]storage.Word{}
		concepts, _, err := addConceptPair(words, nil, "bank", "", input)
		if err == nil {
			t.Errorf("invoer %q: verwacht een fout", input)
		}
		if len(concepts) != 0 || len(words) != 0 {
			t.Errorf("invoer %q: er mag niets toegevoegd zijn", input)
		}
	}
}

func TestAddConceptPairUsesFirstNonEmptyWordForReverse(t *testing.T) {
	concepts, res, err := addConceptPair(map[string]storage.Word{}, nil, "bank", "", ", pad")
	if err != nil {
		t.Fatal(err)
	}
	if res.ReverseSource != "pad" || concepts[1].ID != "c_hu_nl_pad" {
		t.Errorf("verwacht 'pad' als bron, kreeg %q / %s", res.ReverseSource, concepts[1].ID)
	}
}

func TestAddConceptPairDuplicate(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := addConceptPair(words, nil, "bank", "Meubel", "pad")
	after, _, err := addConceptPair(words, concepts, "Bank", "meubel", "kanapé")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("verwacht ErrDuplicate, kreeg %v", err)
	}
	if len(after) != len(concepts) {
		t.Error("bij een duplicaat mag er niets toegevoegd worden")
	}
}

func TestAddConceptPairMergesReverse(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := addConceptPair(words, nil, "tijd", "", "idő")
	concepts, res, err := addConceptPair(words, concepts, "weer", "", "idő")
	if err != nil {
		t.Fatal(err)
	}
	if !res.ReverseMerged {
		t.Error("verwacht dat het HU->NL concept is samengevoegd")
	}
	if len(concepts) != 3 {
		t.Fatalf("verwacht 3 concepten (2x NL->HU, 1x HU->NL), kreeg %d", len(concepts))
	}
	rev := concepts[indexOf(concepts, "c_hu_nl_idő")]
	if !slices.Equal(rev.TargetWordIDs, []string{"w_nl_tijd", "w_nl_weer"}) {
		t.Errorf("HU->NL targets: %v", rev.TargetWordIDs)
	}
}
