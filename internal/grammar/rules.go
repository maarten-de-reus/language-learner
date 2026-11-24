// De regels die je kunt oefenen, met uitleg. Elke regel hoort bij een woordsoort: naamvallen bij
// zelfstandige naamwoorden, vervoegingen bij werkwoorden.

package grammar

import "strings"

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
	pool func(words []Word) []Word // vaste lijst opgavewoorden; nil = je eigen woorden van WordType
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

	r.make = func(word string, _ Person) (Form, bool) {
		if ex, ok := caseExceptions[strings.ToLower(word)][id]; ok {
			return ex, true
		}
		return applySuffix(word, s)
	}
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
		{ID: "kor", Group: "Naamvallen", Name: "-kor", Meaning: "om (tijdstip)", WordType: "zn", Example: "öt → ötkor, éjfél → éjfélkor",
			Explain: "Hoe laat, op welk moment? -kor achter een klokuur of feestdag, voor alle woorden hetzelfde: geen klinkerharmonie en geen verlenging (vacsorakor).\n" +
				"Geoefend met een vaste lijst tijdstippen: dagdelen gaan anders (reggel, délben, este)."},
		{ID: "pers", Group: "Naamvallen", Name: "Naamval met persoon", Meaning: "bij mij, met jou …", WordType: "vnw", Persons: true, Example: "-nál/-nél → nálam, -val/-vel → veled",
			Explain: "Naamvallen gaan niet achter én, te, ő: ze krijgen zelf een persoonsuitgang: -m, -d, –, -unk/-ünk, -tok/-tek, -uk/-ük (nálam, nálad, nála, nálunk, nálatok, náluk).\n" +
				"De stam wisselt per naamval en moet je leren: nek-, nál-, től-, vel-, hozzá-, benn-, belől-, belé-, rajt-, ról-, rá-, ért-."},
		{ID: "t", Group: "Meervoud en lijdend voorwerp", Name: "-t", Meaning: "lijdend voorwerp", WordType: "zn", Example: "asztal → asztalt, ablak → ablakot",
			Explain: "Wat zie of koop je? Het lijdend voorwerp krijgt -t. Na een klinker direct (kávé → kávét, alma → almát). Na j, l, ly, n, ny, r, s, sz, z, zs ook direct (asztalt, pénzt). Na andere medeklinkers met verbindingsklinker -ot, -et of -öt (ablakot, könyvet, gyümölcsöt).\nVeel korte woorden wijken af (ház → házat, ló → lovat): die staan in de uitzonderingen." + harmonyNote},
		{ID: "k", Group: "Meervoud en lijdend voorwerp", Name: "-k", Meaning: "meervoud", WordType: "zn", Example: "alma → almák, ablak → ablakok",
			Explain: "Meervoud: -k. Na een klinker direct (autó → autók, alma → almák). Na een medeklinker met verbindingsklinker -ok, -ek of -ök (ablakok, kertek, gyümölcsök).\nOok hier wijken korte woorden vaak af (ház → házak, madár → madarak)." + harmonyNote},
		{ID: "bir", Group: "Bezittelijke vormen", Name: "Bezittelijk (mijn, jouw …)", Meaning: "mijn huis, jouw huis …", WordType: "zn", Persons: true, Example: "ház → házam, házad, háza",
			Explain: "Eén bezitter, één ding: én -m, te -d, ő -a/-e of -ja/-je, mi -nk/-unk/-ünk, ti -tok/-tek/-tök, ők -uk/-ük of -juk/-jük.\n" +
				"Na een klinker direct, met -á/-é voor een eind-a/-e en altijd een j bij ő en ők: almám, almád, almája, almánk, almátok, almájuk.\n" +
				"Na een medeklinker met verbindingsklinker: asztalom, kertem, gyümölcsöm; woorden die bij -t van stam veranderen doen dat hier ook (házam, lovam, kezem, sarkom).\n" +
				"ő en ők: géén j na s, sz, z, cs, gy, ny, ly, j (pénze, hegye, lánya) en na -ság/-ség; wél een j na twee medeklinkers (kertje, boltja). Daartussen is het per woord (asztala, maar telefonja): die vraagt de app pas als het woord in de lijst staat.\n" +
				"Onregelmatig: fiú → fia, apa → apja, anya → anyja, idő → ideje, szó → szava, út → útja."},
		{ID: "i", Group: "Afleidingen", Name: "-i", Meaning: "uit, van (plaats of tijd)", WordType: "zn", Example: "Budapest → budapesti",
			Explain: "Van een plaats of tijd een bijvoeglijk naamwoord maken: -i, voor alle woorden hetzelfde, zonder verlenging (Amerika → amerikai) en met een kleine letter. Eindigt het woord al op -i, dan komt er niets bij.\n" +
				"Een paar worden korter: este → esti, hét → heti, hónap → havi, Eger → egri.\n" +
				"Geoefend met een vaste lijst plaatsen en tijdwoorden: achter andere woorden betekent -i meestal iets anders (kávéi = zijn koffies). Plaatsnamen hoef je niet als woord te leren."},
		{ID: "szor", Group: "Afleidingen", Name: "-szor/-szer/-ször", Meaning: "… keer", WordType: "telw", Example: "három → háromszor, öt → ötször",
			Explain: "Hoe vaak? Achter een telwoord -szor, -szer of -ször, met klinkerharmonie en zonder verlenging. Kettő wordt kétszer.\n" +
				"Geoefend met de getallen en je eigen telwoorden."},
		{ID: "bb", Group: "Vergelijken", Name: "-bb", Meaning: "groter, mooier (vergrotende trap)", WordType: "bn", Example: "drága → drágább, magas → magasabb",
			Explain: "Vergrotende trap: na een klinker -bb, eind-a/-e wordt á/é (drága → drágább, fekete → feketébb). Na een medeklinker -abb of -ebb (magas → magasabb, kedves → kedvesebb).\n" +
				"Uitzonderingen: jó → jobb, sok → több, kicsi → kisebb, szép → szebb, nagy → nagyobb, öreg → idősebb, nehéz → nehezebb, hosszú → hosszabb, lassú → lassabb, könnyű → könnyebb."},
		{ID: "leg", Group: "Vergelijken", Name: "leg-…-bb", Meaning: "grootst, mooist (overtreffende trap)", WordType: "bn", Example: "nagy → legnagyobb",
			Explain: "Overtreffende trap: leg- voor de vergrotende trap (legdrágább, legjobb). Wat bij -bb onregelmatig is, blijft dat."},
		{ID: "ww-onbep", Group: "Werkwoorden", Name: "Tegenwoordige tijd, onbepaald", Meaning: "ik lees, jij leest …", WordType: "ww", Persons: true, Example: "olvas → olvasok, olvasol",
			Explain: "Uitgangen achter de stam (de 3e persoon enkelvoud, zonder -ik): én -ok/-ek/-ök, te -sz, ő –, mi -unk/-ünk, ti -tok/-tek/-tök, ők -nak/-nek.\n" +
				"Eindigt de stam op s, sz of z, dan wordt 'te' -ol/-el/-öl (olvasol, nézel). Eindigt hij op -ít of twee medeklinkers, dan komt er een verbindingsklinker bij -sz en -nak (tanítasz, mondanak).\n" +
				"-ik-werkwoorden: én op -om/-em/-öm (lakom). Onregelmatig: van, megy, jön en een paar andere." + harmonyNote},
		{ID: "ww-bep", Group: "Werkwoorden", Name: "Tegenwoordige tijd, bepaald", Meaning: "ik lees het …", WordType: "ww", Persons: true, Example: "olvas → olvasom, olvassa",
			Explain: "Met een bepaald lijdend voorwerp (a könyvet): achter: -om, -od, -ja, -juk, -játok, -ják; voor: -em, -ed, -i, -jük, -itek, -ik (rond: -öm, -öd).\n" +
				"Na s, sz en z wordt de j gelijk aan die medeklinker: olvas + ja → olvassa, néz + jük → nézzük. Werkwoorden zonder lijdend voorwerp (megy, lakik) hebben geen bepaalde vervoeging."},
		{ID: "ww-lak", Group: "Werkwoorden", Name: "-lak/-lek", Meaning: "ik … jou", WordType: "ww", Example: "lát → látlak, szeret → szeretlek",
			Explain: "Doe ík iets met jóu, dan is er één vorm: -lak (achter) of -lek (voor) achter de stam: látlak (ik zie je), szeretlek (ik hou van je), várlak (ik wacht op je).\n" +
				"Na -ít of twee medeklinkers met verbindingsklinker: tanítalak, értelek, mondalak. Een stam op -ll krijgt geen derde l: hallak.\n" +
				"Alleen bij werkwoorden met een lijdend voorwerp (niet bij megy, lakik)."},
		{ID: "ww-mult", Group: "Werkwoorden", Name: "Verleden tijd, onbepaald", Meaning: "ik las, jij las …", WordType: "ww", Persons: true, Example: "tanul → tanultam, olvas → olvasott",
			Explain: "Verleden tijd: -t- achter de stam, dan de uitgang: én -am/-em, te -ál/-él, mi -unk/-ünk, ti -atok/-etek, ők -ak/-ek (tanultam, tanultál, tanultunk, tanultatok, tanultak; néztem, néztél).\n" +
				"Na -ít, na medeklinker + t (ért, tart, fest) en bij korte stammen op -t (fut) komt er overal -ott/-ett/-ött tussen: tanítottam, értettem, futottam.\n" +
				"ő: na j, l, ly, n, ny, r alleen -t (tanult, beszélt, várt, kért); anders -ott/-ett/-ött (olvasott, nézett, főzött, mondott, dolgozott).\n" +
				"Onregelmatig: van → volt, megy → ment, jön → jött, eszik → evett, iszik → ivott, tesz → tett, vesz → vett, visz → vitt, alszik → aludt, lát → láttam maar látott."},
		{ID: "ww-mult-bep", Group: "Werkwoorden", Name: "Verleden tijd, bepaald", Meaning: "ik las het …", WordType: "ww", Persons: true, Example: "olvas → olvastam, olvasta",
			Explain: "Zelfde verleden-stuk als bij onbepaald (olvast-, tanított-), met de bepaalde uitgangen: achter -am, -ad, -a, -uk, -átok, -ák (olvastam, olvastad, olvasta); voor -em, -ed, -e, -ük, -étek, -ék (néztem, nézted, nézte). Alleen bij werkwoorden met een lijdend voorwerp."},
		{ID: "ww-fog", Group: "Werkwoorden", Name: "Toekomende tijd (fog)", Meaning: "ik zal lezen …", WordType: "ww", Persons: true, Example: "olvas → olvasni fogok",
			Explain: "fog + infinitief: fogok, fogsz, fog, fogunk, fogtok, fognak met de infinitief ervoor of erna (olvasni fogok / fogok olvasni). Vaak gebruik je gewoon de tegenwoordige tijd met een tijdwoord (holnap olvasok)."},
		{ID: "ww-hat", Group: "Werkwoorden", Name: "-hat/-het", Meaning: "kunnen, mogen", WordType: "ww", Persons: true, Example: "olvas → olvashat, néz → nézhet",
			Explain: "-hat (achter) of -het (voor) achter de stam, daarna vervoegd als gewoon werkwoord: olvashatok, olvashatsz, olvashat, olvashatunk, olvashattok, olvashatnak.\n" +
				"Onregelmatig: van/lesz → lehet, megy → mehet, jön → jöhet, eszik → ehet, iszik → ihat, tesz → tehet, vesz → vehet, visz → vihet, hisz → hihet, alszik → alhat."},
		{ID: "ww-felsz", Group: "Werkwoorden", Name: "Gebiedende wijs", Meaning: "lees!, laten we lezen …", WordType: "ww", Persons: true, Example: "tanul → tanulj!, olvas → olvass!",
			Explain: "Er komt een j tussen stam en uitgang: én -jak/-jek, te -j (of -jál/-jél), ő -jon/-jen/-jön, mi -junk/-jünk, ti -jatok/-jetek, ők -janak/-jenek (tanuljak, tanulj, tanuljon …). Ook na hogy: azt akarom, hogy tanulj.\n" +
				"De j smelt samen met de laatste medeklinker: na s, sz, z wordt hij die medeklinker (olvass, nézz, játssz); klinker + t wordt ss (láss, fizess); -ít en medeklinker + t worden ts (taníts, érts); st wordt ss (fess).\n" +
				"Onregelmatig: van/lesz → legyen, megy → menj, jön → gyere/gyertek, eszik → egyél, iszik → igyál, tesz → tegyél, vesz → vegyél, visz → vigyél, hisz → higgyél."},
		{ID: "ww-felsz-bep", Group: "Werkwoorden", Name: "Gebiedende wijs, bepaald", Meaning: "lees het! …", WordType: "ww", Persons: true, Example: "olvas → olvasd!, néz → nézd!",
			Explain: "Met een bepaald lijdend voorwerp: én -jam/-jem, te kort -d (olvasd, nézd, tanuld, lásd, tanítsd) of lang -jad/-jed, ő -ja/-je, mi -juk/-jük, ti -játok/-jétek, ők -ják/-jék, met dezelfde samensmelting van de j (olvassa, nézze, tanítsa). Onregelmatig: edd, idd, tedd, vedd, vidd, hidd."},
		{ID: "ww-felt", Group: "Werkwoorden", Name: "Voorwaardelijke wijs", Meaning: "ik zou lezen …", WordType: "ww", Persons: true, Example: "olvas → olvasnék, szeret → szeretnék",
			Explain: "De infinitief zonder -ni, dan: én -nék, te -nál/-nél, ő -na/-ne, mi -nánk/-nénk, ti -nátok/-nétek, ők -nának/-nének (olvasnék, olvasnál, olvasna; néznék, néznél, nézne). Let op: én is altijd -nék.\n" +
				"Omdat het uit de infinitief komt, gaan de onregelmatige vanzelf: enni → ennék, inni → innék, lenni → lennék (of volna), menni → mennék, tanítani → tanítanék.\n" +
				"Veel gebruikt om iets beleefd te vragen: szeretnék egy kávét (ik zou graag een koffie willen)."},
		{ID: "ww-felt-bep", Group: "Werkwoorden", Name: "Voorwaardelijke wijs, bepaald", Meaning: "ik zou het lezen …", WordType: "ww", Persons: true, Example: "olvas → olvasnám, olvasná",
			Explain: "Met een bepaald lijdend voorwerp: achter -nám, -nád, -ná, -nánk, -nátok, -nák; voor -ném, -néd, -né, -nénk, -nétek, -nék (olvasnám, olvasná; nézném, nézné)."},
		{ID: "ww-felt-mult", Group: "Werkwoorden", Name: "Voorwaardelijke wijs, verleden", Meaning: "ik zou gelezen hebben …", WordType: "ww", Persons: true, Example: "olvas → olvastam volna",
			Explain: "De verleden tijd met volna erachter, voor alle personen hetzelfde volna: olvastam volna, olvastál volna, olvasott volna (ik zou gelezen hebben …)."},
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
		case "bir":
			r.make = func(w string, p Person) (Form, bool) { return Possessive(w, p) }
		case "i":
			s := caseSuffix{kind: adjective, forms: []string{"i"}}
			r.make = func(w string, _ Person) (Form, bool) { return applySuffix(w, s) }
			r.pool = func(words []Word) []Word { return poolOf(iWords, "i", words) }
		case "kor":
			r.make = func(w string, _ Person) (Form, bool) { return korForm(w) }
			r.pool = func(words []Word) []Word { return poolOf(korWords, "kor", words) }
		case "szor":
			s := caseSuffix{forms: []string{"szor", "szer", "ször"}}
			r.make = func(w string, _ Person) (Form, bool) { return applySuffix(w, s) }
			r.pool = szorPool
		case "bb":
			r.make = func(w string, _ Person) (Form, bool) { return Comparative(w) }
		case "leg":
			r.make = func(w string, _ Person) (Form, bool) { return Superlative(w) }
		case "pers":
			r.make = personalForm
			r.pool = personalPool
		case "ww-onbep":
			r.make = func(w string, p Person) (Form, bool) { return Present(w, p, false) }
		case "ww-bep":
			r.make = func(w string, p Person) (Form, bool) { return Present(w, p, true) }
		case "ww-felt", "ww-felt-bep":
			def := r.ID == "ww-felt-bep"
			r.make = func(w string, p Person) (Form, bool) {
				inf, ok := Infinitive(w)
				if !ok {
					return Form{}, false
				}
				return Conditional(w, inf.Text, p, def)
			}
		case "ww-felt-mult":
			r.make = func(w string, p Person) (Form, bool) { return PastConditional(w, p) }
		case "ww-felsz":
			r.make = func(w string, p Person) (Form, bool) { return Imperative(w, p, false) }
		case "ww-felsz-bep":
			r.make = func(w string, p Person) (Form, bool) { return Imperative(w, p, true) }
		case "ww-lak":
			r.make = func(w string, _ Person) (Form, bool) { return Lak(w) }
		case "ww-mult":
			r.make = func(w string, p Person) (Form, bool) { return Past(w, p, false) }
		case "ww-mult-bep":
			r.make = func(w string, p Person) (Form, bool) { return Past(w, p, true) }
		case "ww-fog":
			r.make = func(w string, p Person) (Form, bool) {
				inf, ok := Infinitive(w)
				if !ok {
					return Form{}, false
				}
				return Future(inf.Text, p)
			}
		case "ww-hat":
			r.make = func(w string, p Person) (Form, bool) { return Potential(w, p) }
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
