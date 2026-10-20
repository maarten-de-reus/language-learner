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
