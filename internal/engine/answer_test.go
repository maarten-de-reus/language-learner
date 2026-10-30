package engine

import "testing"

func TestCheckAnswer(t *testing.T) {
	cases := []struct {
		input      string
		accepted   []string
		correct    bool
		accentOnly bool
	}{
		{"kávé", []string{"kávé"}, true, false},
		{"  KÁVÉ ", []string{"kávé"}, true, false},         // hoofdletters en spaties maken niet uit
		{"jó   napot", []string{"jó napot"}, true, false},  // meerdere spaties tellen als één
		{"kanapé", []string{"pad", "kanapé"}, true, false}, // elk geaccepteerd antwoord telt
		{"kave", []string{"kávé"}, false, true},            // alleen accenten fout
		{"szep", []string{"szép"}, false, true},            // é -> e
		{"ot", []string{"öt"}, false, true},                // ö -> o
		{"oreg", []string{"öreg", "régi"}, false, true},    // ook bij meerdere antwoorden
		{"ő", []string{"ö"}, false, true},                  // ő en ö zijn verschillende letters
		{"thee", []string{"kávé"}, false, false},           // gewoon fout
		{"", []string{"kávé"}, false, false},               // leeg is fout, geen accent-hint
		{"   ", []string{"kávé"}, false, false},            // alleen spaties ook
		{"kávé", nil, false, false},                        // geen antwoorden: altijd fout
		{"één", []string{"een"}, false, true},              // Nederlandse accenten werken ook
	}
	for _, c := range cases {
		correct, accentOnly := CheckAnswer(c.input, c.accepted)
		if correct != c.correct || accentOnly != c.accentOnly {
			t.Errorf("CheckAnswer(%q, %v) = (%v, %v), verwacht (%v, %v)", c.input, c.accepted, correct, accentOnly, c.correct, c.accentOnly)
		}
	}
}

func TestFold(t *testing.T) {
	for in, want := range map[string]string{
		"Idő":           "ido",
		"  Jó   Napot ": "jo napot",
		"ŐSZ":           "osz",
		"tűz":           "tuz",
	} {
		if got := Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, verwacht %q", in, got, want)
		}
	}
}

func TestOptionalParts(t *testing.T) {
	accepted := []string{"(hij/zij) leest"}
	for _, in := range []string{"leest", "hij leest", "zij leest", "hij/zij leest", "(hij/zij) leest", "  Hij   LEEST "} {
		if ok, _ := CheckAnswer(in, accepted); !ok {
			t.Errorf("%q moet goed zijn voor %v", in, accepted)
		}
	}
	for _, in := range []string{"het leest", "hij", "lezen"} {
		if ok, _ := CheckAnswer(in, accepted); ok {
			t.Errorf("%q mag niet goed zijn voor %v", in, accepted)
		}
	}
	if ok, accent := CheckAnswer("eszik", []string{"(ő) eszik"}); !ok || accent {
		t.Error("optioneel deel werkt ook in het Hongaars")
	}
	if ok, accent := CheckAnswer("o eszik", []string{"(ő) eszik"}); ok || !accent {
		t.Error("accent-hint werkt ook voor de varianten")
	}
	for _, in := range []string{"koffie", "de koffie", "koffie zwart", "de koffie zwart"} {
		if ok, _ := CheckAnswer(in, []string{"(de) koffie (zwart)"}); !ok {
			t.Errorf("%q moet goed zijn bij twee optionele delen", in)
		}
	}
	if got := Variants("kávé"); len(got) != 1 || got[0] != "kávé" {
		t.Errorf("zonder haakjes blijft het antwoord zoals het is, kreeg %v", got)
	}
}

func TestPunctuationAndDashes(t *testing.T) {
	cases := []struct {
		input    string
		accepted []string
		correct  bool
	}{
		{"mi", []string{"mi?"}, true},                     // vraagteken aan het eind telt niet
		{"Mi?", []string{"mi"}, true},                     // ook niet als je het wél typt
		{"tessek", []string{"tessék?"}, false},            // accenten tellen nog wel
		{"sorry", []string{"sorry? (wat zegt u?)"}, true}, // optioneel deel en vraagteken
		{"tai-csi doen", []string{"tai–csi doen"}, true},  // lang streepje in het antwoord
		{"tai–csi doen", []string{"tai-csi doen"}, true},  // lang streepje in de invoer
		{"m", []string{"mi?"}, false},
	}
	for _, c := range cases {
		if got, _ := CheckAnswer(c.input, c.accepted); got != c.correct {
			t.Errorf("CheckAnswer(%q, %q) = %v, verwacht %v", c.input, c.accepted, got, c.correct)
		}
	}
	if got := Clean("  tai—csi – doen "); got != "tai-csi - doen" {
		t.Errorf("Clean: %q", got)
	}
}
