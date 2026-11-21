package grammar

import "testing"

func TestPast(t *testing.T) {
	for verb, want := range map[string][6]string{
		"tanul":    {"tanultam", "tanultál", "tanult", "tanultunk", "tanultatok", "tanultak"},
		"beszél":   {"beszéltem", "beszéltél", "beszélt", "beszéltünk", "beszéltetek", "beszéltek"},
		"olvas":    {"olvastam", "olvastál", "olvasott", "olvastunk", "olvastatok", "olvastak"},
		"néz":      {"néztem", "néztél", "nézett", "néztünk", "néztetek", "néztek"},
		"főz":      {"főztem", "főztél", "főzött", "főztünk", "főztetek", "főztek"},
		"mond":     {"mondtam", "mondtál", "mondott", "mondtunk", "mondtatok", "mondtak"},
		"küld":     {"küldtem", "küldtél", "küldött", "küldtünk", "küldtetek", "küldtek"},
		"kezd":     {"kezdtem", "kezdtél", "kezdett", "kezdtünk", "kezdtetek", "kezdtek"},
		"tanít":    {"tanítottam", "tanítottál", "tanított", "tanítottunk", "tanítottatok", "tanítottak"},
		"segít":    {"segítettem", "segítettél", "segített", "segítettünk", "segítettetek", "segítettek"},
		"ért":      {"értettem", "értettél", "értett", "értettünk", "értettetek", "értettek"},
		"tart":     {"tartottam", "tartottál", "tartott", "tartottunk", "tartottatok", "tartottak"},
		"fest":     {"festettem", "festettél", "festett", "festettünk", "festettetek", "festettek"},
		"fut":      {"futottam", "futottál", "futott", "futottunk", "futottatok", "futottak"},
		"fizet":    {"fizettem", "fizettél", "fizetett", "fizettünk", "fizettetek", "fizettek"},
		"dolgozik": {"dolgoztam", "dolgoztál", "dolgozott", "dolgoztunk", "dolgoztatok", "dolgoztak"},
		"ír":       {"írtam", "írtál", "írt", "írtunk", "írtatok", "írtak"},
		"tud":      {"tudtam", "tudtál", "tudott", "tudtunk", "tudtatok", "tudtak"},
		"akar":     {"akartam", "akartál", "akart", "akartunk", "akartatok", "akartak"},
		"fejlődik": {"fejlődtem", "fejlődtél", "fejlődött", "fejlődtünk", "fejlődtetek", "fejlődtek"},
		"van":      {"voltam", "voltál", "volt", "voltunk", "voltatok", "voltak"},
		"megy":     {"mentem", "mentél", "ment", "mentünk", "mentetek", "mentek"},
		"elmegy":   {"elmentem", "elmentél", "elment", "elmentünk", "elmentetek", "elmentek"},
		"eszik":    {"ettem", "ettél", "evett", "ettünk", "ettetek", "ettek"},
		"lát":      {"láttam", "láttál", "látott", "láttunk", "láttatok", "láttak"},
		"játszik":  {"játszottam", "játszottál", "játszott", "játszottunk", "játszottatok", "játszottak"},
		"mosolyog": {"mosolyogtam", "mosolyogtál", "mosolygott", "mosolyogtunk", "mosolyogtatok", "mosolyogtak"},
		"marad":    {"maradtam", "maradtál", "maradt", "maradtunk", "maradtatok", "maradtak"},
	} {
		for p, form := range want {
			if f, ok := Past(verb, Person(p), false); !ok || f.Text != form {
				t.Errorf("%s verleden, %s: %q (ok %v), verwacht %q; %v", verb, PersonLabels[p], f.Text, ok, form, f.Steps)
			}
		}
	}
	for verb, want := range map[string][6]string{
		"olvas": {"olvastam", "olvastad", "olvasta", "olvastuk", "olvastátok", "olvasták"},
		"néz":   {"néztem", "nézted", "nézte", "néztük", "néztétek", "nézték"},
		"ért":   {"értettem", "értetted", "értette", "értettük", "értettétek", "értették"},
		"lát":   {"láttam", "láttad", "látta", "láttuk", "láttátok", "látták"},
		"eszik": {"ettem", "etted", "ette", "ettük", "ettétek", "ették"},
		"vesz":  {"vettem", "vetted", "vette", "vettük", "vettétek", "vették"},
		"iszik": {"ittam", "ittad", "itta", "ittuk", "ittátok", "itták"},
		"ír":    {"írtam", "írtad", "írta", "írtuk", "írtátok", "írták"},
		"tanít": {"tanítottam", "tanítottad", "tanította", "tanítottuk", "tanítottátok", "tanították"},
	} {
		for p, form := range want {
			if f, ok := Past(verb, Person(p), true); !ok || f.Text != form {
				t.Errorf("%s verleden bepaald, %s: %q (ok %v), verwacht %q", verb, PersonLabels[p], f.Text, ok, form)
			}
		}
	}
	for _, verb := range []string{"megy", "lakik", "marad"} {
		if _, ok := Past(verb, Sg1, true); ok {
			t.Errorf("%s heeft geen bepaalde vervoeging", verb)
		}
	}
	if f, ok := Past("enged", Sg3, false); ok {
		t.Errorf("enged (meerlettergrepig op -d, niet in de lijst): verwacht geen vorm, kreeg %s", f.Text)
	}
}

func TestFutureAndPotential(t *testing.T) {
	fog, _ := RuleByID("ww-fog")
	if r, _ := Check("ww-fog", Word{Hungarian: "olvas"}, int(Sg1), "olvasni fogok"); !r.Correct {
		t.Errorf("olvasni fogok: %+v", r)
	}
	if r, _ := Check("ww-fog", Word{Hungarian: "megy"}, int(Pl3), "fognak menni"); !r.Correct {
		t.Errorf("fognak menni (andere volgorde): %+v", r)
	}
	if len(fog.persons(Word{Hungarian: "olvas", Type: "ww"})) != 6 {
		t.Error("fog: alle zes personen")
	}
	for verb, want := range map[string][6]string{
		"olvas":    {"olvashatok", "olvashatsz", "olvashat", "olvashatunk", "olvashattok", "olvashatnak"},
		"néz":      {"nézhetek", "nézhetsz", "nézhet", "nézhetünk", "nézhettek", "nézhetnek"},
		"főz":      {"főzhetek", "főzhetsz", "főzhet", "főzhetünk", "főzhettek", "főzhetnek"},
		"dolgozik": {"dolgozhatok", "dolgozhatsz", "dolgozhat", "dolgozhatunk", "dolgozhattok", "dolgozhatnak"},
		"megy":     {"mehetek", "mehetsz", "mehet", "mehetünk", "mehettek", "mehetnek"},
		"eszik":    {"ehetek", "ehetsz", "ehet", "ehetünk", "ehettek", "ehetnek"},
		"iszik":    {"ihatok", "ihatsz", "ihat", "ihatunk", "ihattok", "ihatnak"},
		"elmegy":   {"elmehetek", "elmehetsz", "elmehet", "elmehetünk", "elmehettek", "elmehetnek"},
	} {
		for p, form := range want {
			if f, ok := Potential(verb, Person(p)); !ok || f.Text != form {
				t.Errorf("%s -hat, %s: %q, verwacht %q", verb, PersonLabels[p], f.Text, form)
			}
		}
	}
	// Afwijkende -hat-vorm bij het woord wordt gewoon vervoegd
	w := Word{Hungarian: "xyz", Type: "ww", Forms: map[string]string{"hat": "abhat"}}
	if r, _ := Check("ww-hat", w, int(Sg1), "abhatok"); !r.Correct {
		t.Errorf("afwijkende -hat-vorm: %+v", r)
	}
}
