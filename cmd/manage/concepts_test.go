package main

import (
	"errors"
	"slices"
	"testing"

	"hungarian-language-learner/internal/storage"
)

// add is een hulpfunctie: parse + addConceptPair, met optionele aanpassing van de invoer
func add(t *testing.T, words map[string]storage.Word, concepts []storage.TranslationConcept, dutch, context, hungarian string, adjust ...func(*newConcept)) ([]storage.TranslationConcept, addResult, error) {
	t.Helper()
	nc, err := parseInput(dutch, context, hungarian)
	if err != nil {
		return concepts, addResult{}, err
	}
	for _, f := range adjust {
		f(&nc)
	}
	return addConceptPair(words, concepts, nc)
}

func TestAddConceptPairCreatesBothDirections(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, res, err := add(t, words, nil, "bank", "Meubel", "pad, kanapé")
	if err != nil {
		t.Fatal(err)
	}
	if len(concepts) != 2 {
		t.Fatalf("verwacht 2 concepten, kreeg %d", len(concepts))
	}
	if got := concepts[0].TargetWordIDs; !slices.Equal(got, []string{"w_hu_pad", "w_hu_kanapé"}) {
		t.Errorf("NL->HU targets: %v", got)
	}
	rev := concepts[1]
	if rev.SourceWordID != "w_hu_pad" || rev.ContextSource != "Meubel" {
		t.Errorf("HU->NL concept moet bron 'pad' en standaard de NL-hint krijgen: %+v", rev)
	}
	if res.ReverseMerged {
		t.Error("ReverseMerged zou false moeten zijn")
	}
}

func TestParseInputRejectsEmptyTranslation(t *testing.T) {
	for _, input := range []string{"", ",", " , ,"} {
		if _, err := parseInput("bank", "", input); err == nil {
			t.Errorf("invoer %q: verwacht een fout", input)
		}
	}
	if _, err := parseInput("  ", "", "pad"); err == nil {
		t.Error("leeg NL-woord: verwacht een fout")
	}
}

func TestFirstNonEmptyWordIsReverseSource(t *testing.T) {
	_, res, err := add(t, map[string]storage.Word{}, nil, "bank", "", ", pad")
	if err != nil {
		t.Fatal(err)
	}
	if res.ReverseSource != "pad" {
		t.Errorf("verwacht 'pad' als bron, kreeg %q", res.ReverseSource)
	}
}

func TestAddConceptPairDuplicate(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := add(t, words, nil, "bank", "Meubel", "pad")
	after, _, err := add(t, words, concepts, "Bank", "meubel", "kanapé")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("verwacht ErrDuplicate, kreeg %v", err)
	}
	if len(after) != len(concepts) {
		t.Error("bij een duplicaat mag er niets toegevoegd worden")
	}
}

// ő -> hij / zij: Hongaars heeft geen geslacht, dus dezelfde betekenis met twee goede antwoorden
func TestSameMeaningMergesReverse(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := add(t, words, nil, "hij", "Pronomen", "ő")
	revID := concepts[1].ID

	concepts, res, err := add(t, words, concepts, "zij", "Pronomen vrouw", "ő", func(nc *newConcept) {
		nc.MergeReverseInto = revID
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.ReverseMerged || len(concepts) != 3 {
		t.Fatalf("verwacht samenvoegen (3 concepten), kreeg %d", len(concepts))
	}
	if got := concepts[indexOf(concepts, revID)].TargetWordIDs; !slices.Equal(got, []string{"w_nl_hij", "w_nl_zij"}) {
		t.Errorf("HU->NL targets: %v", got)
	}
}

// idő -> weer / tijd: verschillende betekenissen, dus aparte concepten met elk een eigen hint
func TestDifferentMeaningCreatesSeparateReverse(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := add(t, words, nil, "weer", "", "idő") // zoals in de bestaande data: zonder hint
	oldID := concepts[1].ID

	concepts, res, err := add(t, words, concepts, "tijd", "Klok", "idő", func(nc *newConcept) {
		nc.ExistingHints = map[string]string{oldID: "buiten ☀️"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReverseMerged || len(concepts) != 4 {
		t.Fatalf("verwacht een apart HU->NL concept (4 concepten), kreeg %d", len(concepts))
	}
	if old := concepts[indexOf(concepts, oldID)]; old.ContextSource != "buiten ☀️" || !slices.Equal(old.TargetWordIDs, []string{"w_nl_weer"}) {
		t.Errorf("bestaand concept moet alleen een hint krijgen: %+v", old)
	}
	if rev := concepts[3]; rev.ContextSource != "Klok" || !slices.Equal(rev.TargetWordIDs, []string{"w_nl_tijd"}) {
		t.Errorf("nieuw HU->NL concept: %+v", rev)
	}
}

func TestReverseHintMustDistinguishMeaning(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := add(t, words, nil, "weer", "Buiten", "idő")

	after, _, err := add(t, words, concepts, "tijd", "buiten", "idő") // zelfde hint als bestaand concept
	if !errors.Is(err, ErrReverseAmbiguous) {
		t.Fatalf("verwacht ErrReverseAmbiguous, kreeg %v", err)
	}
	if len(after) != len(concepts) || len(words) != 2 {
		t.Error("bij een fout mag er niets toegevoegd worden")
	}
}

func TestMergeIntoWrongConceptFails(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := add(t, words, nil, "bank", "", "pad")
	_, _, err := add(t, words, concepts, "weer", "", "idő", func(nc *newConcept) {
		nc.MergeReverseInto = concepts[1].ID // pad, niet idő
	})
	if err == nil {
		t.Fatal("samenvoegen met een concept van een ander woord moet falen")
	}
}

// legacyData bootst de oude data na: genummerde ID's, zoals in woorden.json
func legacyData() (map[string]storage.Word, []storage.TranslationConcept) {
	words := map[string]storage.Word{
		"w_nl_1":  {ID: "w_nl_1", Language: "nl", Text: "hallo"},
		"w_hu_1":  {ID: "w_hu_1", Language: "hu", Text: "szia"},
		"w_nl_28": {ID: "w_nl_28", Language: "nl", Text: "koffie"},
		"w_hu_28": {ID: "w_hu_28", Language: "hu", Text: "kávé"},
	}
	concepts := []storage.TranslationConcept{
		{ID: "c_1", SourceWordID: "w_nl_1", ContextSource: "Groet", TargetWordIDs: []string{"w_hu_1"}},
	}
	return words, concepts
}

func TestDuplicateDetectedAgainstLegacyIDs(t *testing.T) {
	words, concepts := legacyData()
	_, _, err := add(t, words, concepts, "Hallo", " groet ", "szia")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("verwacht ErrDuplicate voor bestaand concept c_1, kreeg %v", err)
	}
	if len(words) != 4 {
		t.Error("bij een duplicaat mogen er geen woorden bijkomen")
	}
}

func TestSameWordDifferentContextIsAllowed(t *testing.T) {
	words, concepts := legacyData()
	if _, _, err := add(t, words, concepts, "hallo", "Telefoon", "halló"); err != nil {
		t.Fatalf("andere context moet mogen, kreeg %v", err)
	}
}

func TestExistingWordsAreReused(t *testing.T) {
	words, concepts := legacyData()
	concepts, _, err := add(t, words, concepts, "Koffie", "Drank", "KÁVÉ")
	if err != nil {
		t.Fatal(err)
	}
	nlhu := concepts[len(concepts)-2]
	if nlhu.SourceWordID != "w_nl_28" || !slices.Equal(nlhu.TargetWordIDs, []string{"w_hu_28"}) {
		t.Errorf("verwacht hergebruik van w_nl_28 en w_hu_28, kreeg %+v", nlhu)
	}
	if len(words) != 4 {
		t.Errorf("er mogen geen nieuwe woorden bijkomen, kreeg %d woorden", len(words))
	}
}

func TestGeneratedIDsDoNotCollide(t *testing.T) {
	words := map[string]storage.Word{
		"w_hu_pad": {ID: "w_hu_pad", Language: "hu", Text: "pád"}, // ID bezet door een ander woord
	}
	concepts := []storage.TranslationConcept{
		{ID: "c_nl_hu_bank", SourceWordID: "w_hu_pad", TargetWordIDs: []string{"w_hu_pad"}}, // ID bezet
	}
	concepts, _, err := add(t, words, concepts, "bank", "", "pad")
	if err != nil {
		t.Fatal(err)
	}
	if words["w_hu_pad_2"].Text != "pad" || words["w_hu_pad"].Text != "pád" {
		t.Errorf("verwacht nieuw woord w_hu_pad_2, woorden: %v", words)
	}
	if concepts[1].ID != "c_nl_hu_bank_2" {
		t.Errorf("verwacht c_nl_hu_bank_2, kreeg %s", concepts[1].ID)
	}
}

func TestExampleStoredOnBothDirections(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, err := add(t, words, nil, "weer", "buiten", "idő", func(nc *newConcept) {
		nc.Example = "Milyen lesz az idő holnap?"
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range concepts {
		if c.Example != "Milyen lesz az idő holnap?" || c.ExampleLang != "hu" {
			t.Errorf("voorbeeldzin ontbreekt op %s: %+v", c.ID, c)
		}
	}
}

func TestMergeKeepsExistingExample(t *testing.T) {
	words := map[string]storage.Word{}
	concepts, _, _ := add(t, words, nil, "hij", "", "ő", func(nc *newConcept) { nc.Example = "Ő a barátom." })
	revID := concepts[1].ID
	concepts, _, _ = add(t, words, concepts, "zij", "vrouw", "ő", func(nc *newConcept) {
		nc.MergeReverseInto = revID
		nc.Example = "Ő a nővérem."
	})
	if got := concepts[indexOf(concepts, revID)].Example; got != "Ő a barátom." {
		t.Errorf("bestaande voorbeeldzin mag niet overschreven worden, kreeg %q", got)
	}
}
