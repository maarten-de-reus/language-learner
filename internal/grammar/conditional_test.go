package grammar

import (
	"slices"
	"testing"
)

func TestConditional(t *testing.T) {
	felt, _ := RuleByID("ww-felt")
	bep, _ := RuleByID("ww-felt-bep")
	for verb, want := range map[string][6]string{
		"olvas":    {"olvasnék", "olvasnál", "olvasna", "olvasnánk", "olvasnátok", "olvasnának"},
		"néz":      {"néznék", "néznél", "nézne", "néznénk", "néznétek", "néznének"},
		"szeret":   {"szeretnék", "szeretnél", "szeretne", "szeretnénk", "szeretnétek", "szeretnének"},
		"főz":      {"főznék", "főznél", "főzne", "főznénk", "főznétek", "főznének"},
		"tanít":    {"tanítanék", "tanítanál", "tanítana", "tanítanánk", "tanítanátok", "tanítanának"},
		"mond":     {"mondanék", "mondanál", "mondana", "mondanánk", "mondanátok", "mondanának"},
		"ért":      {"érteném"[:0] + "értenék", "értenél", "értene", "értenénk", "értenétek", "értenének"},
		"van":      {"lennék", "lennél", "lenne", "lennénk", "lennétek", "lennének"},
		"megy":     {"mennék", "mennél", "menne", "mennénk", "mennétek", "mennének"},
		"eszik":    {"ennék", "ennél", "enne", "ennénk", "ennétek", "ennének"},
		"iszik":    {"innék", "innál", "inna", "innánk", "innátok", "innának"},
		"jön":      {"jönnék", "jönnél", "jönne", "jönnénk", "jönnétek", "jönnének"},
		"alszik":   {"aludnék", "aludnál", "aludna", "aludnánk", "aludnátok", "aludnának"},
		"dolgozik": {"dolgoznék", "dolgoznál", "dolgozna", "dolgoznánk", "dolgoznátok", "dolgoznának"},
		"ír":       {"írnék", "írnál", "írna", "írnánk", "írnátok", "írnának"},
		"elmegy":   {"elmennék", "elmennél", "elmenne", "elmennénk", "elmennétek", "elmennének"},
	} {
		for p, form := range want {
			if f, ok := felt.MakeFor(Word{Hungarian: verb, Type: "ww"}, Person(p)); !ok || f.Text != form {
				t.Errorf("%s voorwaardelijk, %s: %q (ok %v), verwacht %q; %v", verb, PersonLabels[p], f.Text, ok, form, f.Steps)
			}
		}
	}
	for verb, want := range map[string][6]string{
		"olvas": {"olvasnám", "olvasnád", "olvasná", "olvasnánk", "olvasnátok", "olvasnák"},
		"néz":   {"nézném", "néznéd", "nézné", "néznénk", "néznétek", "néznék"},
		"eszik": {"enném", "ennéd", "enné", "ennénk", "ennétek", "ennék"},
	} {
		for p, form := range want {
			if f, ok := bep.MakeFor(Word{Hungarian: verb, Type: "ww"}, Person(p)); !ok || f.Text != form {
				t.Errorf("%s voorwaardelijk bepaald, %s: %q (ok %v), verwacht %q", verb, PersonLabels[p], f.Text, ok, form)
			}
		}
	}
	if f, _ := felt.MakeFor(Word{Hungarian: "van", Type: "ww"}, Sg3); !slices.Contains(f.Also, "volna") {
		t.Errorf("van: lenne of volna, kreeg %v", f.Also)
	}
	if f, _ := felt.MakeFor(Word{Hungarian: "dolgozik", Type: "ww"}, Sg1); !slices.Contains(f.Also, "dolgoznám") {
		t.Errorf("dolgozik: dolgoznék of dolgoznám, kreeg %v", f.Also)
	}
	if _, ok := bep.MakeFor(Word{Hungarian: "megy", Type: "ww"}, Sg1); ok {
		t.Error("megy heeft geen bepaalde vorm")
	}
	if r, _ := Check("ww-felt-mult", Word{Hungarian: "olvas", Type: "ww"}, int(Sg1), "olvastam volna"); !r.Correct {
		t.Errorf("olvastam volna: %+v", r)
	}
	if r, _ := Check("ww-felt-mult", Word{Hungarian: "megy", Type: "ww"}, int(Pl3), "mentek volna"); !r.Correct {
		t.Errorf("mentek volna: %+v", r)
	}
	// Afwijkende infinitief bij het woord
	w := Word{Hungarian: "xyz", Type: "ww", Forms: map[string]string{"inf": "abcni"}}
	if f, _ := felt.MakeFor(w, Sg3); f.Text != "abcne" && f.Text != "abcna" {
		t.Errorf("afwijkende infinitief: %+v", f)
	}
}
