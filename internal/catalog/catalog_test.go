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

// szia -> hallo / hoi: echte synoniemen, dus één concept met twee goede antwoorden
func TestSameMeaningMergesReverse(t *testing.T) {
	lib := newLib()
	add(t, lib, "hallo", "Groet", "szia")
	revID := lib.Concepts[1].ID

	res, err := add(t, lib, "hoi", "Groet informeel", "szia", func(nc *NewPair) { nc.MergeReverseInto = revID })
	if err != nil {
		t.Fatal(err)
	}
	if !res.ReverseMerged || len(lib.Concepts) != 3 {
		t.Fatalf("verwacht samenvoegen (3 concepten), kreeg %d", len(lib.Concepts))
	}
	if got := concept(t, lib, revID).TargetWordIDs; !slices.Equal(got, []string{"w_nl_hallo", "w_nl_hoi"}) {
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
	add(t, lib, "hallo", "", "szia", func(nc *NewPair) { nc.Example = "Szia, hogy vagy?" })
	revID := lib.Concepts[1].ID
	add(t, lib, "hoi", "informeel", "szia", func(nc *NewPair) {
		nc.MergeReverseInto = revID
		nc.Example = "Szia, Anna!"
	})
	if got := concept(t, lib, revID).Example; got != "Szia, hogy vagy?" {
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

func TestEntriesSortedWithMeanings(t *testing.T) {
	lib := newLib()
	add(t, lib, "weer", "buiten", "idő")
	add(t, lib, "tijd", "klok", "idő")
	add(t, lib, "appel", "", "alma")

	entries := Entries(lib)
	if len(entries) != 6 {
		t.Fatalf("verwacht 6 entries, kreeg %d", len(entries))
	}
	var prompts []string
	for _, e := range entries {
		prompts = append(prompts, e.Prompt)
	}
	// "idő" sorteert als "ido", tussen "appel"/"alma" en "tijd"/"weer"
	if want := []string{"alma", "appel", "idő", "idő", "tijd", "weer"}; !slices.Equal(prompts, want) {
		t.Errorf("volgorde: %v, verwacht %v", prompts, want)
	}
	idő := entries[2]
	if idő.Meanings != 2 || idő.From != "hu" || idő.To != "nl" || idő.Context != "buiten" || idő.Interval != 1 {
		t.Errorf("entry idő: %+v", idő)
	}
	if entries[0].Meanings != 1 {
		t.Errorf("alma heeft één betekenis, kreeg %d", entries[0].Meanings)
	}
}

func TestUpdateChangesHintAnswersAndExample(t *testing.T) {
	lib := newLib()
	add(t, lib, "bank", "Meubel", "pad")
	id := lib.Concepts[0].ID
	step, next := lib.Concepts[0].StepIndex, lib.Concepts[0].NextReview

	err := Update(lib, id, Edit{Context: " zitmeubel ", Answers: []string{"pad", " kanapé", "PAD", ""}, Example: "Leülök a padra."})
	if err != nil {
		t.Fatal(err)
	}
	c := concept(t, lib, id)
	if c.ContextSource != "zitmeubel" || c.Example != "Leülök a padra." || c.ExampleLang != "hu" {
		t.Errorf("hint/voorbeeld niet aangepast: %+v", c)
	}
	if got := TargetTexts(lib, c); !slices.Equal(got, []string{"pad", "kanapé"}) {
		t.Errorf("antwoorden: %v (dubbele en lege moeten eruit)", got)
	}
	if lib.Words[c.TargetWordIDs[1]].Language != "hu" {
		t.Error("nieuw antwoord moet in de doeltaal zijn")
	}
	if c.StepIndex != step || !c.NextReview.Equal(next) {
		t.Error("bewerken mag de voortgang niet veranderen")
	}

	if err := Update(lib, id, Edit{Context: "zitmeubel", Answers: []string{"pad"}}); err != nil {
		t.Fatal(err)
	}
	if c := concept(t, lib, id); c.Example != "" || c.ExampleLang != "" {
		t.Errorf("lege voorbeeldzin moet de zin verwijderen: %+v", c)
	}
}

func TestUpdateValidation(t *testing.T) {
	lib := newLib()
	add(t, lib, "weer", "buiten", "idő")
	add(t, lib, "tijd", "klok", "idő")
	tijdRev := lib.Concepts[3].ID
	before := lib.Clone()

	if err := Update(lib, "bestaat-niet", Edit{Answers: []string{"x"}}); !errors.Is(err, ErrNotFound) {
		t.Errorf("verwacht ErrNotFound, kreeg %v", err)
	}
	if err := Update(lib, tijdRev, Edit{Context: "klok", Answers: []string{" ", ""}}); !errors.Is(err, ErrNoAnswers) {
		t.Errorf("verwacht ErrNoAnswers, kreeg %v", err)
	}
	if err := Update(lib, tijdRev, Edit{Context: "Buiten", Answers: []string{"tijd"}}); !errors.Is(err, ErrContextTaken) {
		t.Errorf("hint van de andere betekenis: verwacht ErrContextTaken, kreeg %v", err)
	}
	if len(lib.Words) != len(before.Words) || concept(t, lib, tijdRev).ContextSource != "klok" {
		t.Error("bij een fout mag er niets veranderen")
	}

	// Een hint die alleen bij een ander bronwoord of de andere richting voorkomt is geen probleem
	nlWeer := lib.Concepts[0].ID
	if err := Update(lib, nlWeer, Edit{Context: "klok", Answers: []string{"idő"}}); err != nil {
		t.Errorf("NL->HU 'weer' mag hint 'klok' hebben (andere richting/bron), kreeg %v", err)
	}
}

func TestDeleteSingleDirection(t *testing.T) {
	lib := newLib()
	add(t, lib, "koffie", "", "kávé")
	nlhu, hunl := lib.Concepts[0].ID, lib.Concepts[1].ID

	res, err := Delete(lib, nlhu, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Deleted, []string{nlhu}) || len(lib.Concepts) != 1 || lib.Concepts[0].ID != hunl {
		t.Errorf("alleen %s mag weg, kreeg %+v / %v", nlhu, res, lib.Concepts)
	}
	if len(lib.Words) != 2 {
		t.Error("woorden van de overgebleven richting moeten blijven")
	}
}

func TestDeleteWithReverse(t *testing.T) {
	lib := newLib()
	add(t, lib, "koffie", "", "kávé")
	add(t, lib, "thee", "", "tea")

	res, err := Delete(lib, lib.Concepts[0].ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Deleted) != 2 || len(res.Updated) != 0 || len(lib.Concepts) != 2 {
		t.Errorf("verwacht beide richtingen van koffie weg, kreeg %+v", res)
	}
	if _, ok := lib.Words["w_nl_koffie"]; ok {
		t.Error("ongebruikte woorden moeten opgeruimd worden")
	}
	if _, ok := lib.Words["w_nl_thee"]; !ok {
		t.Error("woorden van andere concepten moeten blijven")
	}
}

// szia -> hallo, hoi is de omgekeerde richting van zowel hallo -> szia als hoi -> szia
func TestDeleteKeepsSharedReverse(t *testing.T) {
	lib := newLib()
	add(t, lib, "hallo", "Groet", "szia")
	shared := lib.Concepts[1].ID
	add(t, lib, "hoi", "informeel", "szia", func(nc *NewPair) { nc.MergeReverseInto = shared })
	hallo := lib.Concepts[0].ID

	if got := Entries(lib); len(got[0].Reverse) == 0 && len(got[1].Reverse) == 0 {
		t.Error("Entries moet de omgekeerde richting aangeven")
	}

	res, err := Delete(lib, hallo, true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Deleted, []string{hallo}) || !slices.Equal(res.Updated, []string{shared}) {
		t.Errorf("verwacht alleen hallo -> szia weg en szia bijgewerkt, kreeg %+v", res)
	}
	if got := TargetTexts(lib, concept(t, lib, shared)); !slices.Equal(got, []string{"hoi"}) {
		t.Errorf("szia moet alleen nog 'hoi' als antwoord hebben, kreeg %v", got)
	}
}

func TestDeleteUnknown(t *testing.T) {
	if _, err := Delete(newLib(), "bestaat-niet", true); !errors.Is(err, ErrNotFound) {
		t.Errorf("verwacht ErrNotFound, kreeg %v", err)
	}
}

// Twee woorden met dezelfde tekst (zoals "zij" = zij enkelvoud en zij meervoud in de oude data):
// bewerken mag het antwoord niet naar het andere woord laten springen
func TestUpdateKeepsExistingAnswerWords(t *testing.T) {
	lib := &storage.Library{
		Words: map[string]storage.Word{
			"w_hu_ok":   {ID: "w_hu_ok", Language: "hu", Text: "ők"},
			"w_nl_15":   {ID: "w_nl_15", Language: "nl", Text: "zij"},
			"w_nl_12_f": {ID: "w_nl_12_f", Language: "nl", Text: "zij"}, // sorteert eerder
		},
		Concepts: []storage.TranslationConcept{
			{ID: "c_hu_nl_ok", SourceWordID: "w_hu_ok", TargetWordIDs: []string{"w_nl_15"}},
		},
	}
	if err := Update(lib, "c_hu_nl_ok", Edit{Answers: []string{"zij", "ze"}, Example: "Ők magyarok."}); err != nil {
		t.Fatal(err)
	}
	got := lib.Concepts[0].TargetWordIDs
	if len(got) != 2 || got[0] != "w_nl_15" || lib.Words[got[1]].Text != "ze" {
		t.Errorf("bestaand antwoord moet w_nl_15 blijven en 'ze' moet erbij komen, kreeg %v", got)
	}
}
