package catalog

import (
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
	results, ok := Import(lib, ParseImport(text), nil)
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
	results, ok := Import(lib.Clone(), ParseImport(text), nil)
	if ok || results[0].Status != StatusChoice || len(results[0].Candidates) != 1 {
		t.Fatalf("verwacht een keuze met 1 bestaande betekenis, kreeg %+v", results)
	}
	revID := results[0].Candidates[0].ID

	// Andere betekenis zonder hint voor het bestaande concept: nog steeds een keuze nodig
	results, ok = Import(lib.Clone(), ParseImport(text), map[int]ImportChoice{1: {}})
	if ok || results[0].Status != StatusChoice {
		t.Errorf("zonder hint voor het bestaande concept moet het geweigerd worden: %+v", results)
	}

	// Andere betekenis met hint
	clone := lib.Clone()
	results, ok = Import(clone, ParseImport(text), map[int]ImportChoice{1: {ExistingHints: map[string]string{revID: "buiten"}}})
	if !ok || results[0].Status != StatusAdded || len(clone.Concepts) != 4 {
		t.Errorf("andere betekenis: %+v, %d concepten", results, len(clone.Concepts))
	}

	// Zelfde betekenis
	clone = lib.Clone()
	results, ok = Import(clone, ParseImport(text), map[int]ImportChoice{1: {MergeInto: revID}})
	if !ok || len(clone.Concepts) != 3 {
		t.Errorf("zelfde betekenis: verwacht 3 concepten, kreeg %d (%+v)", len(clone.Concepts), results)
	}
}

func TestImportSeesEarlierLines(t *testing.T) {
	lib := newLib()
	text := "idő;weer;buiten\nidő;tijd;klok"
	results, ok := Import(lib, ParseImport(text), nil)
	if ok || results[0].Status != StatusAdded || results[1].Status != StatusChoice {
		t.Fatalf("de tweede regel moet de eerste zien: %+v", results)
	}
	if c := results[1].Candidates; len(c) != 1 || c[0].Context != "buiten" {
		t.Errorf("kandidaat moet het net toegevoegde idő (buiten) zijn: %+v", c)
	}
}

func TestImportErrorsBlock(t *testing.T) {
	lib := newLib()
	results, ok := Import(lib, ParseImport("kávé;koffie\nalleen"), nil)
	if ok || results[1].Status != StatusError {
		t.Errorf("een foute regel moet de import blokkeren: %+v", results)
	}
}
