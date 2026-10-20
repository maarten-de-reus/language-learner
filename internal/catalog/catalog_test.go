package catalog

import (
	"errors"
	"slices"
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

func newLib() *storage.Library {
	return &storage.Library{Words: map[string]storage.Word{}}
}

// add is een hulpfunctie: ParseInput + AddPair, met optionele aanpassing van de invoer
func add(t *testing.T, lib *storage.Library, dutch, context, hungarian string, adjust ...func(*NewPair)) (AddResult, error) {
	t.Helper()
	nc, err := ParseInput(dutch, context, hungarian)
	if err != nil {
		return AddResult{}, err
	}
	for _, f := range adjust {
		f(&nc)
	}
	return AddPair(lib, nc)
}

func concept(t *testing.T, lib *storage.Library, id string) storage.TranslationConcept {
	t.Helper()
	i := lib.Index(id)
	if i < 0 {
		t.Fatalf("concept %s niet gevonden", id)
	}
	return lib.Concepts[i]
}

func TestAddPairCreatesBothDirections(t *testing.T) {
	lib := newLib()
	res, err := add(t, lib, "bank", "Meubel", "pad, kanapé")
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Concepts) != 2 {
		t.Fatalf("verwacht 2 concepten, kreeg %d", len(lib.Concepts))
	}
	if got := lib.Concepts[0].TargetWordIDs; !slices.Equal(got, []string{"w_hu_pad", "w_hu_kanapé"}) {
		t.Errorf("NL->HU targets: %v", got)
	}
	rev := lib.Concepts[1]
	if rev.SourceWordID != "w_hu_pad" || rev.ContextSource != "Meubel" {
		t.Errorf("HU->NL concept moet bron 'pad' en standaard de NL-hint krijgen: %+v", rev)
	}
	if res.ReverseMerged {
		t.Error("ReverseMerged zou false moeten zijn")
	}
	for _, c := range lib.Concepts {
		if time.Since(c.NextReview) > time.Minute || c.NextReview.After(time.Now()) {
			t.Errorf("nieuw concept %s moet meteen te oefenen zijn, NextReview %v", c.ID, c.NextReview)
		}
	}
}

func TestParseInputRejectsEmptyTranslation(t *testing.T) {
	for _, input := range []string{"", ",", " , ,"} {
		if _, err := ParseInput("bank", "", input); err == nil {
			t.Errorf("invoer %q: verwacht een fout", input)
		}
	}
	if _, err := ParseInput("  ", "", "pad"); err == nil {
		t.Error("leeg NL-woord: verwacht een fout")
	}
}

func TestFirstNonEmptyWordIsReverseSource(t *testing.T) {
	res, err := add(t, newLib(), "bank", "", ", pad")
	if err != nil {
		t.Fatal(err)
	}
	if res.ReverseSource != "pad" {
		t.Errorf("verwacht 'pad' als bron, kreeg %q", res.ReverseSource)
	}
}

func TestAddPairDuplicate(t *testing.T) {
	lib := newLib()
	add(t, lib, "bank", "Meubel", "pad")
	before := len(lib.Concepts)
	_, err := add(t, lib, "Bank", "meubel", "kanapé")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("verwacht ErrDuplicate, kreeg %v", err)
	}
	if len(lib.Concepts) != before {
		t.Error("bij een duplicaat mag er niets toegevoegd worden")
	}
}

// ő -> hij / zij: Hongaars heeft geen geslacht, dus dezelfde betekenis met twee goede antwoorden
func TestSameMeaningMergesReverse(t *testing.T) {
	lib := newLib()
	add(t, lib, "hij", "Pronomen", "ő")
	revID := lib.Concepts[1].ID

	res, err := add(t, lib, "zij", "Pronomen vrouw", "ő", func(nc *NewPair) { nc.MergeReverseInto = revID })
	if err != nil {
		t.Fatal(err)
	}
	if !res.ReverseMerged || len(lib.Concepts) != 3 {
		t.Fatalf("verwacht samenvoegen (3 concepten), kreeg %d", len(lib.Concepts))
	}
	if got := concept(t, lib, revID).TargetWordIDs; !slices.Equal(got, []string{"w_nl_hij", "w_nl_zij"}) {
		t.Errorf("HU->NL targets: %v", got)
	}
}

// idő -> weer / tijd: verschillende betekenissen, dus aparte concepten met elk een eigen hint
func TestDifferentMeaningCreatesSeparateReverse(t *testing.T) {
	lib := newLib()
	add(t, lib, "weer", "", "idő") // zoals in de bestaande data: zonder hint
	oldID := lib.Concepts[1].ID

	res, err := add(t, lib, "tijd", "Klok", "idő", func(nc *NewPair) {
		nc.ExistingHints = map[string]string{oldID: "buiten ☀️"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReverseMerged || len(lib.Concepts) != 4 {
		t.Fatalf("verwacht een apart HU->NL concept (4 concepten), kreeg %d", len(lib.Concepts))
	}
	if old := concept(t, lib, oldID); old.ContextSource != "buiten ☀️" || !slices.Equal(old.TargetWordIDs, []string{"w_nl_weer"}) {
		t.Errorf("bestaand concept moet alleen een hint krijgen: %+v", old)
	}
	if rev := lib.Concepts[3]; rev.ContextSource != "Klok" || !slices.Equal(rev.TargetWordIDs, []string{"w_nl_tijd"}) {
		t.Errorf("nieuw HU->NL concept: %+v", rev)
	}
}

func TestReverseHintMustDistinguishMeaning(t *testing.T) {
	lib := newLib()
	add(t, lib, "weer", "Buiten", "idő")

	_, err := add(t, lib, "tijd", "buiten", "idő") // zelfde hint als bestaand concept
	if !errors.Is(err, ErrReverseAmbiguous) {
		t.Fatalf("verwacht ErrReverseAmbiguous, kreeg %v", err)
	}
	if len(lib.Concepts) != 2 || len(lib.Words) != 2 {
		t.Error("bij een fout mag er niets toegevoegd worden")
	}
}

func TestMergeIntoWrongConceptFails(t *testing.T) {
	lib := newLib()
	add(t, lib, "bank", "", "pad")
	padReverse := lib.Concepts[1].ID
	_, err := add(t, lib, "weer", "", "idő", func(nc *NewPair) { nc.MergeReverseInto = padReverse })
	if err == nil {
		t.Fatal("samenvoegen met een concept van een ander woord moet falen")
	}
}

func TestExampleStoredOnBothDirections(t *testing.T) {
	lib := newLib()
	_, err := add(t, lib, "weer", "buiten", "idő", func(nc *NewPair) { nc.Example = "Milyen lesz az idő holnap?" })
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range lib.Concepts {
		if c.Example != "Milyen lesz az idő holnap?" || c.ExampleLang != "hu" {
			t.Errorf("voorbeeldzin ontbreekt op %s: %+v", c.ID, c)
		}
	}
}

func TestMergeKeepsExistingExample(t *testing.T) {
	lib := newLib()
	add(t, lib, "hij", "", "ő", func(nc *NewPair) { nc.Example = "Ő a barátom." })
	revID := lib.Concepts[1].ID
	add(t, lib, "zij", "vrouw", "ő", func(nc *NewPair) {
		nc.MergeReverseInto = revID
		nc.Example = "Ő a nővérem."
	})
	if got := concept(t, lib, revID).Example; got != "Ő a barátom." {
		t.Errorf("bestaande voorbeeldzin mag niet overschreven worden, kreeg %q", got)
	}
}

// legacyLib bootst de oude data na: genummerde ID's, zoals in woorden.json
func legacyLib() *storage.Library {
	return &storage.Library{
		Words: map[string]storage.Word{
			"w_nl_1":  {ID: "w_nl_1", Language: "nl", Text: "hallo"},
			"w_hu_1":  {ID: "w_hu_1", Language: "hu", Text: "szia"},
			"w_nl_28": {ID: "w_nl_28", Language: "nl", Text: "koffie"},
			"w_hu_28": {ID: "w_hu_28", Language: "hu", Text: "kávé"},
		},
		Concepts: []storage.TranslationConcept{
			{ID: "c_1", SourceWordID: "w_nl_1", ContextSource: "Groet", TargetWordIDs: []string{"w_hu_1"}},
		},
	}
}

func TestDuplicateDetectedAgainstLegacyIDs(t *testing.T) {
	lib := legacyLib()
	_, err := add(t, lib, "Hallo", " groet ", "szia")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("verwacht ErrDuplicate voor bestaand concept c_1, kreeg %v", err)
	}
	if len(lib.Words) != 4 {
		t.Error("bij een duplicaat mogen er geen woorden bijkomen")
	}
}

func TestSameWordDifferentContextIsAllowed(t *testing.T) {
	if _, err := add(t, legacyLib(), "hallo", "Telefoon", "halló"); err != nil {
		t.Fatalf("andere context moet mogen, kreeg %v", err)
	}
}

func TestExistingWordsAreReused(t *testing.T) {
	lib := legacyLib()
	if _, err := add(t, lib, "Koffie", "Drank", "KÁVÉ"); err != nil {
		t.Fatal(err)
	}
	nlhu := lib.Concepts[len(lib.Concepts)-2]
	if nlhu.SourceWordID != "w_nl_28" || !slices.Equal(nlhu.TargetWordIDs, []string{"w_hu_28"}) {
		t.Errorf("verwacht hergebruik van w_nl_28 en w_hu_28, kreeg %+v", nlhu)
	}
	if len(lib.Words) != 4 {
		t.Errorf("er mogen geen nieuwe woorden bijkomen, kreeg %d woorden", len(lib.Words))
	}
}

func TestGeneratedIDsDoNotCollide(t *testing.T) {
	lib := &storage.Library{
		Words: map[string]storage.Word{
			"w_hu_pad": {ID: "w_hu_pad", Language: "hu", Text: "pád"}, // ID bezet door een ander woord
		},
		Concepts: []storage.TranslationConcept{
			{ID: "c_nl_hu_bank", SourceWordID: "w_hu_pad", TargetWordIDs: []string{"w_hu_pad"}}, // ID bezet
		},
	}
	if _, err := add(t, lib, "bank", "", "pad"); err != nil {
		t.Fatal(err)
	}
	if lib.Words["w_hu_pad_2"].Text != "pad" || lib.Words["w_hu_pad"].Text != "pád" {
		t.Errorf("verwacht nieuw woord w_hu_pad_2, woorden: %v", lib.Words)
	}
	if lib.Concepts[1].ID != "c_nl_hu_bank_2" {
		t.Errorf("verwacht c_nl_hu_bank_2, kreeg %s", lib.Concepts[1].ID)
	}
}
