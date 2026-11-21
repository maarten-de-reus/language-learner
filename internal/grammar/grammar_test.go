package grammar

import (
	"slices"
	"testing"
)

func TestHarmony(t *testing.T) {
	for word, want := range map[string]Harmony{
		"ház": Back, "autó": Back, "papír": Back, "kávé": Back, "kocsi": Back, "híd": Back, "iskola": Back,
		"kert": Front, "kép": Front, "víz": Front, "szék": Front, "kenyér": Front, "Budapest": Front,
		"gyümölcs": Rounded, "sör": Rounded, "kávéfőző": Rounded, "teniszütő": Rounded, "bőrönd": Rounded,
		"számítógép": Front, "szobahőmérséklet": Front, "rendőrség": Front, "fiú": Back, "nagymama": Back,
	} {
		if h, _, why := HarmonyOf(word); h != want {
			t.Errorf("%s: %v (%s), verwacht %v", word, h, why, want)
		}
	}
	if _, mixed, _ := HarmonyOf("hotel"); !mixed {
		t.Error("hotel moet beide varianten toestaan")
	}
}

// check vergelijkt de vormen van een regel met bekende goede vormen
func check(t *testing.T, id string, want map[string]string) {
	t.Helper()
	r, ok := RuleByID(id)
	if !ok {
		t.Fatalf("regel %s bestaat niet", id)
	}
	for word, form := range want {
		f, ok := r.Make(word, Sg3)
		if !ok {
			t.Errorf("%s + %s: geen vorm, verwacht %s", word, r.Name, form)
			continue
		}
		if f.Text != form && !slices.Contains(f.Also, form) {
			t.Errorf("%s + %s: %s, verwacht %s (stappen: %v)", word, r.Name, f.Text, form, f.Steps)
		}
	}
}

func TestCaseSuffixes(t *testing.T) {
	check(t, "ban", map[string]string{"ház": "házban", "kert": "kertben", "alma": "almában", "kefe": "kefében", "iskola": "iskolában",
		"autó": "autóban", "gyümölcs": "gyümölcsben", "papír": "papírban", "kávé": "kávéban", "Budapest": "Budapesten"[:0] + "Budapestben",
		"számítógép": "számítógépben", "hotel": "hotelban", "szoba": "szobában", "víz": "vízben", "híd": "hídban"})
	check(t, "ba", map[string]string{"ház": "házba", "kert": "kertbe", "iskola": "iskolába", "bolt": "boltba", "mozi": "moziba"})
	check(t, "bol", map[string]string{"ház": "házból", "kert": "kertből", "iskola": "iskolából", "bolt": "boltból", "üzlet": "üzletből"})
	check(t, "on", map[string]string{"asztal": "asztalon", "szék": "széken", "föld": "földön", "kávé": "kávén", "tábla": "táblán",
		"ház": "házon", "híd": "hídon", "ló": "lovon", "sarok": "sarkon", "Budapest": "Budapesten", "hét": "héten", "út": "úton", "posta": "postán"})
	check(t, "ra", map[string]string{"asztal": "asztalra", "szék": "székre", "posta": "postára", "piac": "piacra"})
	check(t, "rol", map[string]string{"asztal": "asztalról", "szék": "székről", "könyv": "könyvről"})
	check(t, "nal", map[string]string{"orvos": "orvosnál", "fodrász": "fodrásznál", "barát": "barátnál", "ablak": "ablaknál", "kapu": "kapunál", "fiú": "fiúnál"})
	check(t, "hoz", map[string]string{"orvos": "orvoshoz", "tanár": "tanárhoz", "kert": "kerthez", "főnök": "főnökhöz", "sör": "sörhöz",
		"ablak": "ablakhoz", "szomszéd": "szomszédhoz", "fiú": "fiúhoz", "rendőrség": "rendőrséghez", "kávéfőző": "kávéfőzőhöz", "nagymama": "nagymamához"})
	check(t, "tol", map[string]string{"orvos": "orvostól", "kert": "kerttől", "szomszéd": "szomszédtól", "anya": "anyától"})
	check(t, "nak", map[string]string{"barát": "barátnak", "gyerek": "gyereknek", "anya": "anyának", "főnök": "főnöknek"})
	check(t, "val", map[string]string{"ház": "házzal", "busz": "busszal", "kanál": "kanállal", "toll": "tollal", "kés": "késsel",
		"kocsi": "kocsival", "kávé": "kávéval", "alma": "almával", "barát": "baráttal", "hegy": "heggyel", "könyv": "könyvvel",
		"tej": "tejjel", "vonat": "vonattal", "villamos": "villamossal", "gyerek": "gyerekkel", "nő": "nővel", "fiú": "fiúval"})
	check(t, "va", map[string]string{"víz": "vízzé", "jég": "jéggé", "tanár": "tanárrá", "kő": "kővé"})
	check(t, "ig", map[string]string{"busz": "buszig", "posta": "postáig", "este": "estéig", "hétfő": "hétfőig"})
	check(t, "ert", map[string]string{"kenyér": "kenyérért", "tej": "tejért", "alma": "almáért"})
	check(t, "kent", map[string]string{"tanár": "tanárként", "orvos": "orvosként", "barát": "barátként"})
	check(t, "i", map[string]string{"Budapest": "budapesti", "London": "londoni", "Amerika": "amerikai", "reggel": "reggeli",
		"Helsinki": "helsinki", "Győr": "győri", "Ausztria": "ausztriai", "falu": "falui"})
}

func TestAccusativeAndPlural(t *testing.T) {
	check(t, "t", map[string]string{"kávé": "kávét", "alma": "almát", "autó": "autót", "asztal": "asztalt", "pénz": "pénzt",
		"ablak": "ablakot", "könyv": "könyvet", "gyümölcs": "gyümölcsöt", "ház": "házat", "ló": "lovat", "madár": "madarat",
		"kenyér": "kenyeret", "víz": "vizet", "kéz": "kezet", "bolt": "boltot", "kert": "kertet", "sör": "sört", "busz": "buszt",
		"film": "filmet", "tanár": "tanárt", "barát": "barátot", "kocsi": "kocsit", "kefe": "kefét", "sarok": "sarkot", "kórház": "kórházat", "munkatárs": "munkatársat", "gitár": "gitárt", "könyvtár": "könyvtárat",
		"belváros": "belvárost", "játszótér": "játszóteret", "munkahely": "munkahelyet", "fotó": "fotót"})
	check(t, "k", map[string]string{"alma": "almák", "autó": "autók", "kefe": "kefék", "ablak": "ablakok", "kert": "kertek",
		"gyümölcs": "gyümölcsök", "ház": "házak", "madár": "madarak", "ló": "lovak", "könyv": "könyvek", "asztal": "asztalok",
		"busz": "buszok", "tanár": "tanárok", "barát": "barátok", "sör": "sörök", "szék": "székek", "orvos": "orvosok", "falu": "falvak", "kórház": "kórházak", "evőkanál": "evőkanalak"})
	// Onbekende korte woorden geven geen vorm: liever niet oefenen dan raden
	if f, ok := accOrPlural("kar", false); ok {
		t.Errorf("kar: verwacht geen vorm (eenlettergrepig, niet in de lijsten), kreeg %s", f.Text)
	}
}

func TestPresentIndefinite(t *testing.T) {
	for verb, want := range map[string][6]string{
		"olvas":      {"olvasok", "olvasol", "olvas", "olvasunk", "olvastok", "olvasnak"},
		"beszél":     {"beszélek", "beszélsz", "beszél", "beszélünk", "beszéltek", "beszélnek"},
		"köszön":     {"köszönök", "köszönsz", "köszön", "köszönünk", "köszöntök", "köszönnek"},
		"tanul":      {"tanulok", "tanulsz", "tanul", "tanulunk", "tanultok", "tanulnak"},
		"néz":        {"nézek", "nézel", "néz", "nézünk", "néztek", "néznek"},
		"főz":        {"főzök", "főzöl", "főz", "főzünk", "főztök", "főznek"},
		"tanít":      {"tanítok", "tanítasz", "tanít", "tanítunk", "tanítotok", "tanítanak"},
		"segít":      {"segítek", "segítesz", "segít", "segítünk", "segítetek", "segítenek"},
		"mond":       {"mondok", "mondasz", "mond", "mondunk", "mondtok", "mondanak"},
		"ért":        {"értek", "értesz", "ért", "értünk", "értetek", "értenek"},
		"küld":       {"küldök", "küldesz", "küld", "küldünk", "küldtök", "küldenek"},
		"tart":       {"tartok", "tartasz", "tart", "tartunk", "tartotok", "tartanak"},
		"lakik":      {"lakom", "laksz", "lakik", "lakunk", "laktok", "laknak"},
		"dolgozik":   {"dolgozom", "dolgozol", "dolgozik", "dolgozunk", "dolgoztok", "dolgoznak"},
		"játszik":    {"játszom", "játszol", "játszik", "játszunk", "játszotok", "játszanak"},
		"úszik":      {"úszom", "úszol", "úszik", "úszunk", "úsztok", "úsznak"},
		"eszik":      {"eszem", "eszel", "eszik", "eszünk", "esztek", "esznek"},
		"iszik":      {"iszom", "iszol", "iszik", "iszunk", "isztok", "isznak"},
		"ír":         {"írok", "írsz", "ír", "írunk", "írtok", "írnak"},
		"van":        {"vagyok", "vagy", "van", "vagyunk", "vagytok", "vannak"},
		"megy":       {"megyek", "mész", "megy", "megyünk", "mentek", "mennek"},
		"elmegy":     {"elmegyek", "elmész", "elmegy", "elmegyünk", "elmentek", "elmennek"},
		"jön":        {"jövök", "jössz", "jön", "jövünk", "jöttök", "jönnek"},
		"alszik":     {"alszom", "alszol", "alszik", "alszunk", "alszotok", "alszanak"},
		"közlekedik": {"közlekedem", "közlekedsz", "közlekedik", "közlekedünk", "közlekedtek", "közlekednek"},
		"fejlődik":   {"fejlődöm", "fejlődsz", "fejlődik", "fejlődünk", "fejlődtök", "fejlődnek"},
		"tesz":       {"teszek", "teszel", "tesz", "teszünk", "tesztek", "tesznek"},
		"kezd":       {"kezdek", "kezdesz", "kezd", "kezdünk", "kezdtek", "kezdenek"},
		"fest":       {"festek", "festesz", "fest", "festünk", "festetek", "festenek"},
	} {
		for p, form := range want {
			f, ok := Present(verb, Person(p), false)
			if !ok || f.Text != form {
				t.Errorf("%s, %s: %q (ok %v), verwacht %q; stappen %v", verb, PersonLabels[p], f.Text, ok, form, f.Steps)
			}
		}
	}
	// Niet zeker: lát -> láttok maar süt -> sütötök
	if f, ok := Present("süt", Pl2, false); ok {
		t.Errorf("süt, ti: verwacht geen vorm, kreeg %s", f.Text)
	}
}

func TestPresentDefinite(t *testing.T) {
	for verb, want := range map[string][6]string{
		"olvas":   {"olvasom", "olvasod", "olvassa", "olvassuk", "olvassátok", "olvassák"},
		"mond":    {"mondom", "mondod", "mondja", "mondjuk", "mondjátok", "mondják"},
		"játszik": {"játszom", "játszod", "játssza", "játsszuk", "játsszátok", "játsszák"},
		"lát":     {"látom", "látod", "látja", "látjuk", "látjátok", "látják"},
		"néz":     {"nézem", "nézed", "nézi", "nézzük", "nézitek", "nézik"},
		"ért":     {"értem", "érted", "érti", "értjük", "értitek", "értik"},
		"köszön":  {"köszönöm", "köszönöd", "köszöni", "köszönjük", "köszönitek", "köszönik"},
		"tanít":   {"tanítom", "tanítod", "tanítja", "tanítjuk", "tanítjátok", "tanítják"},
		"eszik":   {"eszem", "eszed", "eszi", "esszük", "eszitek", "eszik"},
		"iszik":   {"iszom", "iszod", "issza", "isszuk", "isszátok", "isszák"},
		"főz":     {"főzöm", "főzöd", "főzi", "főzzük", "főzitek", "főzik"},
		"hoz":     {"hozom", "hozod", "hozza", "hozzuk", "hozzátok", "hozzák"},
	} {
		for p, form := range want {
			f, ok := Present(verb, Person(p), true)
			if !ok || f.Text != form {
				t.Errorf("%s bepaald, %s: %q (ok %v), verwacht %q; stappen %v", verb, PersonLabels[p], f.Text, ok, form, f.Steps)
			}
		}
	}
	for _, verb := range []string{"megy", "lakik", "van", "dolgozik"} {
		if _, ok := Present(verb, Sg1, true); ok {
			t.Errorf("%s heeft geen bepaalde vervoeging", verb)
		}
	}
}

func TestInfinitive(t *testing.T) {
	for verb, want := range map[string]string{
		"olvas": "olvasni", "lakik": "lakni", "tanít": "tanítani", "mond": "mondani", "ért": "érteni", "tanul": "tanulni",
		"beszél": "beszélni", "játszik": "játszani", "dolgozik": "dolgozni", "úszik": "úszni", "van": "lenni", "megy": "menni",
		"elmegy": "elmenni", "jön": "jönni", "eszik": "enni", "iszik": "inni", "alszik": "aludni", "tesz": "tenni", "visz": "vinni",
		"lát": "látni", "segít": "segíteni", "fest": "festeni", "küld": "küldeni", "köszön": "köszönni", "fürdik": "fürdeni", "hall": "hallani",
	} {
		f, ok := Infinitive(verb)
		if !ok || f.Text != want {
			t.Errorf("%s: %q (ok %v), verwacht %q; stappen %v", verb, f.Text, ok, want, f.Steps)
		}
	}
}

func TestRulesComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rules {
		if seen[r.ID] || r.make == nil || r.Explain == "" || r.WordType == "" {
			t.Errorf("regel %+v: dubbel of onvolledig", r.ID)
		}
		seen[r.ID] = true
	}
}

func TestMoreRules(t *testing.T) {
	kor, _ := RuleByID("kor")
	for word, want := range map[string]string{"öt": "ötkor", "éjfél": "éjfélkor", "tizenkettő": "tizenkettőkor", "vacsora": "vacsorakor", "hét": "hétkor"} {
		if r, ok := Check("kor", kor.Lookup(nil, word), 0, want); !ok || !r.Correct {
			t.Errorf("%s -kor: %+v, verwacht %s", word, r.Form, want)
		}
	}
	szor, _ := RuleByID("szor")
	for word, want := range map[string]string{"egy": "egyszer", "kettő": "kétszer", "három": "háromszor", "öt": "ötször",
		"tíz": "tízszer", "húsz": "húszszor", "ezer": "ezerszer", "sok": "sokszor", "több": "többször", "néhány": "néhányszor"} {
		if r, ok := Check("szor", szor.Lookup(nil, word), 0, want); !ok || !r.Correct {
			t.Errorf("%s -szor: %+v, verwacht %s", word, r.Form, want)
		}
	}
	if own := szor.Applicable([]Word{{Hungarian: "huszonöt", Type: "telw"}, {Hungarian: "fél", Type: "telw"}}); !slices.ContainsFunc(own, func(w Word) bool { return w.Hungarian == "huszonöt" }) ||
		slices.ContainsFunc(own, func(w Word) bool { return w.Hungarian == "fél" }) {
		t.Error("eigen telwoorden wel, fél niet")
	}
	for adj, want := range map[string]string{"drága": "drágább", "fekete": "feketébb", "magas": "magasabb", "kedves": "kedvesebb",
		"olcsó": "olcsóbb", "késő": "később", "jó": "jobb", "nagy": "nagyobb", "szép": "szebb", "kicsi": "kisebb",
		"hosszú": "hosszabb", "fiatal": "fiatalabb", "rövid": "rövidebb", "zöld": "zöldebb", "gyors": "gyorsabb", "öreg": "idősebb"} {
		if f, ok := Comparative(adj); !ok || f.Text != want {
			t.Errorf("%s: %q, verwacht %q", adj, f.Text, want)
		}
		if f, ok := Superlative(adj); !ok || f.Text != "leg"+want {
			t.Errorf("%s overtreffend: %q", adj, f.Text)
		}
	}
	for _, adj := range []string{"cuki", "magyar", "kisebb", "bal"} {
		if f, ok := Comparative(adj); ok {
			t.Errorf("%s: verwacht geen vorm, kreeg %s", adj, f.Text)
		}
	}
	pers, _ := RuleByID("pers")
	if r, ok := Check("pers", pers.Lookup(nil, "-nál/-nél"), int(Sg1), "nálam"); !ok || !r.Correct {
		t.Errorf("nálam: %+v", r)
	}
	if r, ok := Check("pers", pers.Lookup(nil, "-hoz/-hez/-höz"), int(Pl3), "hozzájuk"); !ok || !r.Correct {
		t.Errorf("hozzájuk: %+v", r)
	}
	if n := len(pers.Applicable(nil)); n != 12 {
		t.Errorf("persoonsvormen: %d naamvallen", n)
	}
}
