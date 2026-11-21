package grammar

import "testing"

func TestAnalyze(t *testing.T) {
	cases := []struct {
		w           Word
		input, rule string
		person      string
	}{
		{Word{Hungarian: "olvas", Type: "ww"}, "olvasok", "ww-onbep", "én (ik)"},
		{Word{Hungarian: "olvas", Type: "ww"}, "olvassa", "ww-bep", "ő (hij/zij)"},
		{Word{Hungarian: "olvas", Type: "ww"}, "olvasni", "ww-inf", ""},
		{Word{Hungarian: "ház", Type: "zn"}, "házban", "ban", ""},
		{Word{Hungarian: "ház", Type: "zn"}, "hazat", "t", ""}, // zonder accenten herkend
		{Word{Hungarian: "nagy", Type: "bn"}, "nagyobb", "bb", ""},
	}
	for _, c := range cases {
		m, ok := Analyze(c.w, c.input)
		if !ok || m.Rule != c.rule || m.Person != c.person {
			t.Errorf("%s als %s: %+v %v, verwacht %s %s", c.input, c.w.Hungarian, m, ok, c.rule, c.person)
		}
	}
	if _, ok := Analyze(Word{Hungarian: "ház", Type: "zn"}, "kert"); ok {
		t.Error("kert is geen vorm van ház")
	}
	if _, ok := Analyze(Word{Hungarian: "ház", Type: "zn"}, "ház"); ok {
		t.Error("de basisvorm zelf telt niet")
	}

}
