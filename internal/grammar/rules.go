// De regels die je kunt oefenen, met uitleg. Elke regel hoort bij een woordsoort: naamvallen bij
// zelfstandige naamwoorden, vervoegingen bij werkwoorden.

package grammar

// Rule is een regel uit de oefenlijst
type Rule struct {
	ID       string `json:"id"`
	Group    string `json:"group"`    // Naamvallen, Meervoud en lijdend voorwerp, Werkwoorden, Afleidingen
	Name     string `json:"name"`     // -ban/-ben
	Meaning  string `json:"meaning"`  // in (waar?)
	Explain  string `json:"explain"`  // de regel in een paar zinnen
	WordType string `json:"wordType"` // op welke woordsoort de regel werkt (zn, ww)
	Persons  bool   `json:"persons"`  // per persoon (werkwoorden)
	Example  string `json:"example"`  // vast voorbeeld: ház → házban

	make func(word string, p Person) (Form, bool)
}

const harmonyNote = "\nWelke variant: kijk naar de klinkers. Achterklinkers (a á o ó u ú) → de eerste variant; voorklinkers (e é i í) → de tweede; " +
	"bij drie varianten krijgen woorden die eindigen op een ronde klinker (ö ő ü ű) de derde. " +
	"Na een achterklinker tellen i, í en é niet mee: papír en kávé zijn achter."

const lengthenNote = "\nEindigt het woord op -a of -e, dan wordt dat -á of -é (alma → almában, kefe → kefében)."

func caseRule(id, name, meaning, example, explain string, s caseSuffix) Rule {
	if len(s.forms) > 1 {
		explain += harmonyNote
	}
	if s.lengthen {
		explain += lengthenNote
	}
	r := Rule{ID: id, Group: "Naamvallen", Name: name, Meaning: meaning, Explain: explain, WordType: "zn", Example: example}

	r.make = func(word string, _ Person) (Form, bool) { return applySuffix(word, s) }
	return r
}

// Rules is de volledige lijst, in de volgorde van het overzicht
var Rules = func() []Rule {
	rules := []Rule{
		caseRule("ban", "-ban/-ben", "in (waar?)", "ház → házban", "Waar is iets? Binnen in iets: -ban of -ben.",
			caseSuffix{forms: []string{"ban", "ben"}, lengthen: true}),
		caseRule("ba", "-ba/-be", "in, naar binnen (waarheen?)", "ház → házba", "Waarheen? Naar binnen: -ba of -be.",
			caseSuffix{forms: []string{"ba", "be"}, lengthen: true}),
		caseRule("bol", "-ból/-ből", "uit (waarvandaan?)", "ház → házból", "Waarvandaan? Van binnen naar buiten: -ból of -ből.",
			caseSuffix{forms: []string{"ból", "ből"}, lengthen: true}),
		{ID: "on", Group: "Naamvallen", Name: "-n/-on/-en/-ön", Meaning: "op (waar?)", WordType: "zn", Example: "asztal → asztalon",
			Explain: "Waar? Op iets: na een klinker alleen -n (kávé → kávén), na een medeklinker -on, -en of -ön." + harmonyNote + lengthenNote},
		caseRule("ra", "-ra/-re", "op, naar (waarheen?)", "asztal → asztalra", "Waarheen? Op iets (er bovenop): -ra of -re.",
			caseSuffix{forms: []string{"ra", "re"}, lengthen: true}),
		caseRule("rol", "-ról/-ről", "van … af, over", "asztal → asztalról", "Waarvandaan? Van de bovenkant af, en ook 'over' (een boek over …): -ról of -ről.",
			caseSuffix{forms: []string{"ról", "ről"}, lengthen: true}),
		caseRule("nal", "-nál/-nél", "bij (waar?)", "orvos → orvosnál", "Waar? Bij iemand of iets: -nál of -nél.",
			caseSuffix{forms: []string{"nál", "nél"}, lengthen: true}),
		caseRule("hoz", "-hoz/-hez/-höz", "naar (toe)", "orvos → orvoshoz", "Waarheen? Naar iemand of iets toe: -hoz, -hez of -höz.",
			caseSuffix{forms: []string{"hoz", "hez", "höz"}, lengthen: true}),
		caseRule("tol", "-tól/-től", "van (vandaan)", "orvos → orvostól", "Waarvandaan? Van iemand of iets weg: -tól of -től.",
			caseSuffix{forms: []string{"tól", "től"}, lengthen: true}),
		caseRule("nak", "-nak/-nek", "aan, voor (wie?)", "barát → barátnak", "Aan of voor wie? Meewerkend voorwerp: -nak of -nek. Ook bij 'hebben' (nekem van).",
			caseSuffix{forms: []string{"nak", "nek"}, lengthen: true}),
		caseRule("val", "-val/-vel", "met", "ház → házzal", "Met iets of iemand: -val of -vel. Na een medeklinker valt de v weg en wordt de laatste medeklinker dubbel (busz → busszal, kanál → kanállal); is die al dubbel, dan blijft het bij twee (toll → tollal).",
			caseSuffix{forms: []string{"val", "vel"}, kind: assimilate, lengthen: true}),
		caseRule("va", "-vá/-vé", "tot (worden)", "jég → jéggé", "Ergens in veranderen: -vá of -vé, met dezelfde v-regel als -val/-vel (víz → vízzé).",
			caseSuffix{forms: []string{"vá", "vé"}, kind: assimilate, lengthen: true}),
		caseRule("ig", "-ig", "tot (aan)", "busz → buszig", "Tot waar of tot wanneer: -ig, voor alle woorden hetzelfde.",
			caseSuffix{forms: []string{"ig"}, lengthen: true}),
		caseRule("ert", "-ért", "voor, om (iets te halen)", "kenyér → kenyérért", "Waarvoor? Voor iets (om het te halen) of in ruil voor: -ért, voor alle woorden hetzelfde.",
			caseSuffix{forms: []string{"ért"}, lengthen: true}),
		caseRule("kent", "-ként", "als", "tanár → tanárként", "In de rol van: -ként, voor alle woorden hetzelfde.",
			caseSuffix{forms: []string{"ként"}, lengthen: true}),
		{ID: "t", Group: "Meervoud en lijdend voorwerp", Name: "-t", Meaning: "lijdend voorwerp", WordType: "zn", Example: "asztal → asztalt, ablak → ablakot",
			Explain: "Wat zie of koop je? Het lijdend voorwerp krijgt -t. Na een klinker direct (kávé → kávét, alma → almát). Na j, l, ly, n, ny, r, s, sz, z, zs ook direct (asztalt, pénzt). Na andere medeklinkers met verbindingsklinker -ot, -et of -öt (ablakot, könyvet, gyümölcsöt).\nVeel korte woorden wijken af (ház → házat, ló → lovat): die staan in de uitzonderingen." + harmonyNote},
		{ID: "k", Group: "Meervoud en lijdend voorwerp", Name: "-k", Meaning: "meervoud", WordType: "zn", Example: "alma → almák, ablak → ablakok",
			Explain: "Meervoud: -k. Na een klinker direct (autó → autók, alma → almák). Na een medeklinker met verbindingsklinker -ok, -ek of -ök (ablakok, kertek, gyümölcsök).\nOok hier wijken korte woorden vaak af (ház → házak, madár → madarak)." + harmonyNote},
		{ID: "i", Group: "Afleidingen", Name: "-i", Meaning: "uit, van (plaats of tijd)", WordType: "zn", Example: "Budapest → budapesti",
			Explain: "Van een plaats of tijd een bijvoeglijk naamwoord maken: -i, voor alle woorden hetzelfde, zonder verlenging (Amerika → amerikai) en met een kleine letter. Eindigt het woord al op -i, dan komt er niets bij.\n" +
				"Een paar worden korter: este → esti, hét → heti, hónap → havi, Eger → egri.\n" +
				"Geoefend met een vaste lijst plaatsen en tijdwoorden: achter andere woorden betekent -i meestal iets anders (kávéi = zijn koffies). Plaatsnamen hoef je niet als woord te leren."},
		{ID: "ww-onbep", Group: "Werkwoorden", Name: "Tegenwoordige tijd, onbepaald", Meaning: "ik lees, jij leest …", WordType: "ww", Persons: true, Example: "olvas → olvasok, olvasol",
			Explain: "Uitgangen achter de stam (de 3e persoon enkelvoud, zonder -ik): én -ok/-ek/-ök, te -sz, ő –, mi -unk/-ünk, ti -tok/-tek/-tök, ők -nak/-nek.\n" +
				"Eindigt de stam op s, sz of z, dan wordt 'te' -ol/-el/-öl (olvasol, nézel). Eindigt hij op -ít of twee medeklinkers, dan komt er een verbindingsklinker bij -sz en -nak (tanítasz, mondanak).\n" +
				"-ik-werkwoorden: én op -om/-em/-öm (lakom). Onregelmatig: van, megy, jön en een paar andere." + harmonyNote},
		{ID: "ww-bep", Group: "Werkwoorden", Name: "Tegenwoordige tijd, bepaald", Meaning: "ik lees het …", WordType: "ww", Persons: true, Example: "olvas → olvasom, olvassa",
			Explain: "Met een bepaald lijdend voorwerp (a könyvet): achter: -om, -od, -ja, -juk, -játok, -ják; voor: -em, -ed, -i, -jük, -itek, -ik (rond: -öm, -öd).\n" +
				"Na s, sz en z wordt de j gelijk aan die medeklinker: olvas + ja → olvassa, néz + jük → nézzük. Werkwoorden zonder lijdend voorwerp (megy, lakik) hebben geen bepaalde vervoeging."},
		{ID: "ww-inf", Group: "Werkwoorden", Name: "Infinitief", Meaning: "lezen, wonen …", WordType: "ww", Example: "olvas → olvasni, tanít → tanítani",
			Explain: "Stam + -ni (olvasni, lakni). Na -ít of twee medeklinkers komt er een verbindingsklinker: -ani/-eni (tanítani, mondani, érteni). -ni zelf doet niet mee aan klinkerharmonie.\n" +
				"Onregelmatig: van/lesz → lenni, megy → menni, eszik → enni, iszik → inni, tesz → tenni, vesz → venni, visz → vinni, hisz → hinni, alszik → aludni."},
	}
	for i := range rules {
		r := &rules[i]
		switch r.ID {
		case "on":
			s := caseSuffix{kind: superessive, lengthen: true, forms: []string{"on", "en", "ön"}}
			r.make = func(w string, _ Person) (Form, bool) { return superessiveFor(w, s) }
		case "t":
			r.make = func(w string, _ Person) (Form, bool) { return accOrPlural(w, false) }
		case "k":
			r.make = func(w string, _ Person) (Form, bool) { return accOrPlural(w, true) }
		case "i":
			s := caseSuffix{kind: adjective, forms: []string{"i"}}
			r.make = func(w string, _ Person) (Form, bool) { return applySuffix(w, s) }
		case "ww-onbep":
			r.make = func(w string, p Person) (Form, bool) { return Present(w, p, false) }
		case "ww-bep":
			r.make = func(w string, p Person) (Form, bool) { return Present(w, p, true) }
		case "ww-inf":
			r.make = func(w string, _ Person) (Form, bool) { return Infinitive(w) }
		}
	}
	return rules
}()

// RuleByID zoekt een regel
func RuleByID(id string) (Rule, bool) {
	for _, r := range Rules {
		if r.ID == id {
			return r, true
		}
	}
	return Rule{}, false
}

// Make maakt de vorm van word volgens deze regel (p alleen bij werkwoorden)
func (r Rule) Make(word string, p Person) (Form, bool) {
	return r.make(word, p)
}
