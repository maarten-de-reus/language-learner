package grammar

import (
	"slices"
	"testing"
)

func TestImperative(t *testing.T) {
	for verb, want := range map[string][6]string{
		"tanul":    {"tanuljak", "tanulj", "tanuljon", "tanuljunk", "tanuljatok", "tanuljanak"},
		"olvas":    {"olvassak", "olvass", "olvasson", "olvassunk", "olvassatok", "olvassanak"},
		"néz":      {"nézzek", "nézz", "nézzen", "nézzünk", "nézzetek", "nézzenek"},
		"főz":      {"főzzek", "főzz", "főzzön", "főzzünk", "főzzetek", "főzzenek"},
		"lát":      {"lássak", "láss", "lásson", "lássunk", "lássatok", "lássanak"},
		"fizet":    {"fizessek", "fizess", "fizessen", "fizessünk", "fizessetek", "fizessenek"},
		"tanít":    {"tanítsak", "taníts", "tanítson", "tanítsunk", "tanítsatok", "tanítsanak"},
		"ért":      {"értsek", "érts", "értsen", "értsünk", "értsetek", "értsenek"},
		"tart":     {"tartsak", "tarts", "tartson", "tartsunk", "tartsatok", "tartsanak"},
		"fest":     {"fessek", "fess", "fessen", "fessünk", "fessetek", "fessenek"},
		"mond":     {"mondjak", "mondj", "mondjon", "mondjunk", "mondjatok", "mondjanak"},
		"ír":       {"írjak", "írj", "írjon", "írjunk", "írjatok", "írjanak"},
		"játszik":  {"játsszak", "játssz", "játsszon", "játsszunk", "játsszatok", "játsszanak"},
		"dolgozik": {"dolgozzak", "dolgozz", "dolgozzon", "dolgozzunk", "dolgozzatok", "dolgozzanak"},
		"fut":      {"fussak", "fuss", "fusson", "fussunk", "fussatok", "fussanak"},
		"van":      {"legyek", "legyél", "legyen", "legyünk", "legyetek", "legyenek"},
		"megy":     {"menjek", "menj", "menjen", "menjünk", "menjetek", "menjenek"},
		"elmegy":   {"elmenjek", "elmenj", "elmenjen", "elmenjünk", "elmenjetek", "elmenjenek"},
		"jön":      {"jöjjek", "gyere", "jöjjön", "jöjjünk", "gyertek", "jöjjenek"},
		"eszik":    {"egyek", "egyél", "egyen", "együnk", "egyetek", "egyenek"},
		"iszik":    {"igyak", "igyál", "igyon", "igyunk", "igyatok", "igyanak"},
		"fejlődik": {"fejlődjek", "fejlődj", "fejlődjön", "fejlődjünk", "fejlődjetek", "fejlődjenek"},
		"fürdik":   {"fürödjek", "fürödj", "fürödjön", "fürödjünk", "fürödjetek", "fürödjenek"},
		"áll":      {"álljak", "állj", "álljon", "álljunk", "álljatok", "álljanak"},
	} {
		for p, form := range want {
			if f, ok := Imperative(verb, Person(p), false); !ok || f.Text != form {
				t.Errorf("%s gebiedend, %s: %q (ok %v), verwacht %q; %v", verb, PersonLabels[p], f.Text, ok, form, f.Steps)
			}
		}
	}
	for verb, want := range map[string][6]string{
		"olvas": {"olvassam", "olvasd", "olvassa", "olvassuk", "olvassátok", "olvassák"},
		"néz":   {"nézzem", "nézd", "nézze", "nézzük", "nézzétek", "nézzék"},
		"tanul": {"tanuljam", "tanuld", "tanulja", "tanuljuk", "tanuljátok", "tanulják"},
		"lát":   {"lássam", "lásd", "lássa", "lássuk", "lássátok", "lássák"},
		"tanít": {"tanítsam", "tanítsd", "tanítsa", "tanítsuk", "tanítsátok", "tanítsák"},
		"ért":   {"értsem", "értsd", "értse", "értsük", "értsétek", "értsék"},
		"fest":  {"fessem", "fesd", "fesse", "fessük", "fessétek", "fessék"},
		"mond":  {"mondjam", "mondd", "mondja", "mondjuk", "mondjátok", "mondják"},
		"eszik": {"egyem", "edd", "egye", "együk", "egyétek", "egyék"},
		"visz":  {"vigyem", "vidd", "vigye", "vigyük", "vigyétek", "vigyék"},
	} {
		for p, form := range want {
			if f, ok := Imperative(verb, Person(p), true); !ok || f.Text != form {
				t.Errorf("%s gebiedend bepaald, %s: %q (ok %v), verwacht %q", verb, PersonLabels[p], f.Text, ok, form)
			}
		}
	}
	// Lange vormen en varianten zijn ook goed
	for _, c := range []struct {
		verb string
		p    Person
		def  bool
		also string
	}{{"olvas", Sg2, false, "olvassál"}, {"tanul", Sg2, false, "tanuljál"}, {"olvas", Sg2, true, "olvassad"},
		{"dolgozik", Sg1, false, "dolgozzam"}, {"jön", Sg2, false, "jöjj"}, {"eszik", Sg2, true, "egyed"}} {
		if f, _ := Imperative(c.verb, c.p, c.def); !slices.Contains(f.Also, c.also) {
			t.Errorf("%s %s: %v moet %q bevatten", c.verb, PersonLabels[c.p], f.Also, c.also)
		}
	}
	for _, verb := range []string{"megy", "lakik", "van"} {
		if _, ok := Imperative(verb, Sg2, true); ok {
			t.Errorf("%s heeft geen bepaalde gebiedende wijs", verb)
		}
	}
}
