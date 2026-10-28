package catalog

import (
	"slices"
	"strings"
	"testing"
)

func TestParseImport(t *testing.T) {
	text := "hongaars;nederlands;context;voorbeeldzin\r\n" +
		"olvas;leest;werkwoord · 3e persoon enkelvoud;Ő sokat olvas.\n" +
		"  pad , kanapé ; bank ;meubel\n" + // spaties mogen
		"\n# opmerking\n" +
		"kávé;koffie\n" + // context en voorbeeld optioneel
		"alleen\n" +
		"a;b;c;d;e\n" +
		";leeg\n" +
		"egy;een, één\n"

	lines := ParseImport(text)
	if len(lines) != 7 {
		t.Fatalf("verwacht 7 regels (kopregel, lege regel en opmerking overgeslagen), kreeg %d: %+v", len(lines), lines)
	}
	if l := lines[0]; l.No != 2 || l.Hungarian != "olvas" || l.Dutch != "leest" || l.Example != "Ő sokat olvas." || l.Err != "" {
		t.Errorf("regel 2: %+v", l)
	}
	if l := lines[1]; l.Hungarian != "pad , kanapé" || l.Dutch != "bank" || l.Context != "meubel" || l.Example != "" {
		t.Errorf("spaties rond ; moeten weg: %+v", l)
	}
	if l := lines[2]; l.Dutch != "koffie" || l.Context != "" || l.Err != "" {
		t.Errorf("twee kolommen: %+v", l)
	}
	for i, wantErr := range map[int]string{3: "te weinig", 4: "te veel", 5: "Hongaars woord ontbreekt", 6: "meerdere Nederlandse"} {
		if !strings.Contains(lines[i].Err, wantErr) {
			t.Errorf("regel %d: verwacht fout %q, kreeg %+v", lines[i].No, wantErr, lines[i])
		}
	}
}

func TestImportAddsAndSkipsDuplicates(t *testing.T) {
	lib := legacyLib() // bevat hallo/Groet -> szia en de woorden koffie, kávé
	text := "szia;hallo;Groet\nkávé;koffie;Drank;Kérek egy kávét.\nnégy;vier;Getal"
	results, ok := Import(lib, ParseImport(text), nil, nil)
	if !ok {
		t.Fatalf("verwacht geen fouten, kreeg %+v", results)
	}
	if results[0].Status != StatusDuplicate || results[1].Status != StatusAdded || results[2].Status != StatusAdded {
		t.Errorf("statussen: %+v", results)
	}
	if len(lib.Concepts) != 1+4 {
		t.Errorf("verwacht 4 nieuwe concepten (2 woorden x 2 richtingen), kreeg %d", len(lib.Concepts)-1)
	}
	koffie := lib.Concepts[1]
	if koffie.Example != "Kérek egy kávét." || koffie.ContextSource != "Drank" || !koffie.FirstReview.IsZero() {
		t.Errorf("koffie: %+v (moet nieuw zijn, met voorbeeldzin)", koffie)
	}
}

func TestImportAsksForMeaning(t *testing.T) {
	lib := newLib()
	add(t, lib, "weer", "", "idő") // bestaand, zonder hint

	text := "idő;tijd;klok"
	results, ok := Import(lib.Clone(), ParseImport(text), nil, nil)
	if ok || results[0].Status != StatusChoice || len(results[0].Candidates) != 1 {
		t.Fatalf("verwacht een keuze met 1 bestaande betekenis, kreeg %+v", results)
	}
	revID := results[0].Candidates[0].ID

	// Andere betekenis zonder hint voor het bestaande concept: nog steeds een keuze nodig
	results, ok = Import(lib.Clone(), ParseImport(text), map[int]ImportChoice{1: {Reverse: true}}, nil)
	if ok || results[0].Status != StatusChoice {
		t.Errorf("zonder hint voor het bestaande concept moet het geweigerd worden: %+v", results)
	}

	// Andere betekenis met hint
	clone := lib.Clone()
	results, ok = Import(clone, ParseImport(text), map[int]ImportChoice{1: {Reverse: true, ExistingHints: map[string]string{revID: "buiten"}}}, nil)
	if !ok || results[0].Status != StatusAdded || len(clone.Concepts) != 4 {
		t.Errorf("andere betekenis: %+v, %d concepten", results, len(clone.Concepts))
	}

	// Zelfde betekenis
	clone = lib.Clone()
	results, ok = Import(clone, ParseImport(text), map[int]ImportChoice{1: {Reverse: true, MergeInto: revID}}, nil)
	if !ok || len(clone.Concepts) != 3 {
		t.Errorf("zelfde betekenis: verwacht 3 concepten, kreeg %d (%+v)", len(clone.Concepts), results)
	}
}

func TestImportSeesEarlierLines(t *testing.T) {
	lib := newLib()
	text := "idő;weer;buiten\nidő;tijd;klok"
	results, ok := Import(lib, ParseImport(text), nil, nil)
	if ok || results[0].Status != StatusAdded || results[1].Status != StatusChoice {
		t.Fatalf("de tweede regel moet de eerste zien: %+v", results)
	}
	if c := results[1].Candidates; len(c) != 1 || c[0].Context != "buiten" {
		t.Errorf("kandidaat moet het net toegevoegde idő (buiten) zijn: %+v", c)
	}
}

func TestImportErrorsBlock(t *testing.T) {
	lib := newLib()
	results, ok := Import(lib, ParseImport("kávé;koffie\nalleen"), nil, nil)
	if ok || results[1].Status != StatusError {
		t.Errorf("een foute regel moet de import blokkeren: %+v", results)
	}
}

func TestImportAsksForDutchMeaning(t *testing.T) {
	lib := newLib()
	add(t, lib, "bank", "meubel", "pad")
	nlhu := lib.Concepts[0].ID

	// kanapé bij bank (meubel): eerst kiezen of het een extra antwoord of een nieuwe betekenis is
	text := "kanapé;bank;meubel"
	results, ok := Import(lib.Clone(), ParseImport(text), nil, nil)
	if ok || results[0].Status != StatusChoice || len(results[0].ForwardCandidates) != 1 {
		t.Fatalf("verwacht een keuze voor het Nederlandse woord, kreeg %+v", results)
	}

	// Extra antwoord: geen nieuw NL->HU concept, wel kanapé -> bank
	clone := lib.Clone()
	results, ok = Import(clone, ParseImport(text), map[int]ImportChoice{1: {Forward: &ForwardChoice{Extend: nlhu}}}, nil)
	if !ok || results[0].Status != StatusAdded || len(clone.Concepts) != 3 {
		t.Fatalf("aanvullen: %+v, %d concepten", results, len(clone.Concepts))
	}
	if got := TargetTexts(clone, clone.Concepts[clone.Index(nlhu)]); !slices.Equal(got, []string{"pad", "kanapé"}) {
		t.Errorf("bank (meubel) moet pad en kanapé accepteren, kreeg %v", got)
	}

	// Nieuwe betekenis met dezelfde hint: een open keuze, op te lossen met een eigen hint in het voorbeeld
	results, _ = Import(lib.Clone(), ParseImport(text), map[int]ImportChoice{1: {Forward: &ForwardChoice{}}}, nil)
	if results[0].Status != StatusChoice {
		t.Errorf("nieuwe betekenis met dezelfde hint moet een open keuze zijn, kreeg %+v", results[0])
	}
	hint := "zitbank"
	clone = lib.Clone()
	results, ok = Import(clone, ParseImport(text), map[int]ImportChoice{1: {Forward: &ForwardChoice{}, Context: &hint}}, nil)
	if !ok || results[0].Context != "zitbank" || len(ForwardCandidates(clone, "bank")) != 2 {
		t.Errorf("met een eigen hint in het voorbeeld moet het lukken: %+v", results[0])
	}

	// Nieuwe betekenis met een andere hint
	clone = lib.Clone()
	results, ok = Import(clone, ParseImport("bank;bank;geld"), map[int]ImportChoice{1: {Forward: &ForwardChoice{}}}, nil)
	if !ok || len(ForwardCandidates(clone, "bank")) != 2 {
		t.Errorf("nieuwe betekenis bank (geld): %+v", results)
	}
}

func TestImportExtendWithNothingNewIsSkipped(t *testing.T) {
	lib := newLib()
	add(t, lib, "koffie", "", "kávé")
	results, ok := Import(lib, ParseImport("kávé;koffie;drank"), map[int]ImportChoice{1: {Forward: &ForwardChoice{Extend: lib.Concepts[0].ID}}}, nil)
	if !ok || results[0].Status != StatusDuplicate {
		t.Errorf("niets nieuws om aan te vullen: verwacht overgeslagen, kreeg %+v", results[0])
	}
}

// De situatie van cuki: schattig -> aranyos bestaat zonder hint, de regel heeft geen context
func TestImportNewMeaningWithInlineHints(t *testing.T) {
	lib := newLib()
	add(t, lib, "schattig", "", "aranyos")
	old := lib.Concepts[0].ID
	text := "cuki;schattig"

	results, _ := Import(lib.Clone(), ParseImport(text), map[int]ImportChoice{1: {Forward: &ForwardChoice{}}}, nil)
	if results[0].Status != StatusChoice {
		t.Fatalf("zonder hints: verwacht een open keuze (geen fout), kreeg %+v", results[0])
	}

	hint := "spreektaal"
	clone := lib.Clone()
	results, ok := Import(clone, ParseImport(text), map[int]ImportChoice{1: {
		Forward: &ForwardChoice{Hints: map[string]string{old: "lief"}},
		Context: &hint,
	}}, nil)
	if !ok {
		t.Fatalf("met beide hints moet het lukken: %+v", results)
	}
	if c := clone.Concepts[clone.Index(old)]; c.ContextSource != "lief" || !slices.Equal(TargetTexts(clone, c), []string{"aranyos"}) {
		t.Errorf("bestaand: verwacht schattig (lief) -> aranyos, kreeg %+v", c)
	}
	var cuki []string
	for _, i := range ForwardCandidates(clone, "schattig") {
		if c := clone.Concepts[i]; c.ContextSource == "spreektaal" {
			cuki = TargetTexts(clone, c)
		}
	}
	if !slices.Equal(cuki, []string{"cuki"}) {
		t.Errorf("verwacht schattig (spreektaal) -> cuki, kreeg %v", cuki)
	}
}

func TestImportReportsSynonymCards(t *testing.T) {
	lib := newLib()
	add(t, lib, "tijd", "klok", "idő")
	results, ok := Import(lib, ParseImport("pad,kanapé;bank;meubel\nóra,idő;uur"), nil, nil)
	if !ok {
		t.Fatalf("import mislukt: %+v", results)
	}
	if r := results[0]; !slices.Equal(r.ExtraReverse, []string{"kanapé"}) || r.SkippedReverse != nil {
		t.Errorf("regel 1: verwacht een kaart voor kanapé, kreeg %+v", r)
	}
	if r := results[1]; r.ExtraReverse != nil || !slices.Equal(r.SkippedReverse, []string{"idő"}) {
		t.Errorf("regel 2: idő heeft al een andere betekenis, kreeg %+v", r)
	}
	if len(ReverseCandidates(lib, "kanapé")) != 1 {
		t.Error("kanapé -> bank moet bestaan")
	}
}

func TestImportExistingWordGetsGroups(t *testing.T) {
	lib := newLib()
	add(t, lib, "oud", "dingen", "régi")
	add(t, lib, "oud", "mensen, dieren", "öreg", func(nc *NewPair) { nc.ForwardHints = map[string]string{lib.Concepts[0].ID: "dingen"} })

	// Zonder hint, met een andere hint of met precies de hint: allemaal het bestaande woord
	results, ok := Import(lib, ParseImport("régi;oud\nöreg;oud;ouderwets\nrégi;oud;dingen\nfiatal;jong\nrégi,öreg;oud"), nil, []string{"Week 2"})
	if !ok {
		t.Fatalf("import mag niet blokkeren: %+v", results)
	}
	want := []struct {
		status, existing string
		grouped          bool
	}{
		{StatusDuplicate, "oud (dingen) → régi", true},
		{StatusDuplicate, "oud (mensen, dieren) → öreg", true},
		{StatusDuplicate, "oud (dingen) → régi", false}, // zat er net al in
		{StatusAdded, "", false},
		{StatusDuplicate, "oud (dingen) → régi en oud (mensen, dieren) → öreg", false}, // beide betekenissen samen
	}
	for i, w := range want {
		if r := results[i]; r.Status != w.status || r.Existing != w.existing || r.Grouped != w.grouped {
			t.Errorf("regel %d: verwacht %+v, kreeg %+v", i+1, w, r)
		}
	}
	for _, c := range lib.Concepts {
		if !slices.Contains(c.Groups, "Week 2") {
			t.Errorf("%s moet in Week 2 zitten (beide richtingen), groepen %v", c.ID, c.Groups)
		}
	}
}
