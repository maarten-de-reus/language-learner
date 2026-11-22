// Bezittelijke vormen (birtokos személyjelek), één bezitter en één ding: házam, házad, háza,
// házunk, házatok, házuk. Het lastige deel is de 3e persoon: háza maar kertje, almája. Wat de
// regels daarover niet zeker weten, wordt niet gevraagd (de andere personen wel).

package grammar

import "strings"

// possIrregular: woorden die hun eigen weg gaan
var possIrregular = map[string][6]string{
	"fiú":  {"fiam", "fiad", "fia", "fiunk", "fiatok", "fiuk"},
	"falu": {"falum", "falud", "faluja", "falunk", "falutok", "falujuk"},
	"szó":  {"szavam", "szavad", "szava", "szavunk", "szavatok", "szavuk"},
	"idő":  {"időm", "időd", "ideje", "időnk", "időtök", "idejük"},
	"apa":  {"apám", "apád", "apja", "apánk", "apátok", "apjuk"},
	"anya": {"anyám", "anyád", "anyja", "anyánk", "anyátok", "anyjuk"},
	"út":   {"utam", "utad", "útja", "utunk", "utatok", "útjuk"},
	"kút":  {"kutam", "kutad", "kútja", "kutunk", "kutatok", "kútjuk"},
	"híd":  {"hidam", "hidad", "hídja", "hidunk", "hidatok", "hídjuk"},
}

// possJ en possNoJ: woorden waarvan de 3e persoon -ja/-je resp. -a/-e is, waar de regel het niet
// zeker weet (ook als laatste deel van een samenstelling: tankönyv als könyv)
var possJ = map[string]bool{}
var possNoJ = map[string]bool{}

func init() {
	for _, w := range []string{"föld", "hold",
		// dagen van de week
		"hétfő", "kedd", "szerda", "csütörtök", "péntek", "szombat", "vasárnap", "nap", "hónap",
		"alak", "banán", "barát", "címzett", "diák", "dollár", "dzsip", "délelőtt", "délután", "ebéd", "fesztivál",
		"fotel", "gitár", "kabát", "laptop", "telefon", "motor", "nadrág", "oroszlán", "papír", "pingvin", "program",
		"rag", "sofőr", "stadion", "szomszéd", "tyúk", "vulkán", "xilofon", "zsiráf", "óceán", "ügyvéd"} {
		possJ[w] = true
	}
	for _, w := range []string{"ablak", "asztal", "beteg", "bor", "citrom", "cím", "dokumentum", "kép", "gép", "főnök",
		"gyerek", "jog", "játék", "kollégium", "kritérium", "környék", "múzeum", "nyelvtan", "pincér", "pályaudvar",
		"reggel", "rendőr", "sor", "szeminárium", "szám", "szék", "szín", "sör", "tanár", "templom", "tenger",
		"húg", "áram", "szer", "év", "üveg"} {
		possNoJ[w] = true
	}
}

// Na deze medeklinkers geen j: pénze, hegye, lánya, kukaca
var noJConsonants = []string{"s", "sz", "z", "zs", "c", "cs", "dz", "dzs", "gy", "ny", "ty", "ly", "j"}

// possTakesJ: krijgt de 3e persoon een j (kertje) of niet (háza)? certain is false als de regels
// het niet weten.
func possTakesJ(w string) (j bool, why string, certain bool) {
	if _, _, ok := lookupSuffix(possJ, w); ok {
		return true, w + " krijgt -ja/-je (staat in de lijst)", true
	}
	if _, _, ok := lookupSuffix(possNoJ, w); ok {
		return false, w + " krijgt -a/-e zonder j (staat in de lijst)", true
	}
	cons, double := finalConsonant(w)
	if contains(noJConsonants, cons) {
		return false, "na " + cons + " komt er geen j: -a/-e", true
	}
	if strings.HasSuffix(w, "ság") || strings.HasSuffix(w, "ség") {
		return false, "na -ság/-ség geen j", true
	}
	before := strings.TrimSuffix(w, cons)
	if cons == "t" && syllables(w) > 1 && strings.ContainsRune("aeoöuü", lastRune(before)) {
		return false, "meerlettergrepig op korte klinker + t: geen j (feladata, mondata)", true
	}
	if !double && before != "" && !isVowel(lastRune(before)) {
		if cons == "v" {
			return false, "na -v geen j (könyve, nyelve)", true
		}
		return true, "na twee medeklinkers komt er een j: -ja/-je (kertje, boltja)", true
	}
	return false, "", false
}

// Possessive: de bezittelijke vorm voor één bezitter (én … ők) en één ding
func Possessive(word string, p Person) (Form, bool) {
	w := strings.ToLower(strings.TrimSpace(word))
	if w == "" || strings.ContainsAny(w, " -") {
		return Form{}, false
	}
	if key, forms, ok := lookupSuffix(possIrregular, w); ok {
		prefix := strings.TrimSuffix(w, key)
		return Form{Text: prefix + forms[p], Steps: []string{key + " is een uitzondering: " + strings.Join(forms[:], ", ")}}, true
	}
	h, _, why := HarmonyOf(w)
	steps := []string{"klinkers van " + w + ": " + why}
	two := func(b, f string) string { return pick(h, b, f) }

	// Stammen die veranderen (ház -> háza-, ló -> lov-, sarok -> sark-): zoals bij het lijdend voorwerp.
	// Een samenstelling volgt de klinkers van het laatste deel (játszótér -> játszótere).
	var p0, link string
	exception := false
	if key, ex, ok := lookupSuffix(nounExceptions, w); ok {
		acc := strings.TrimSuffix(w, key) + ex.acc
		if r := []rune(acc); len(r) > 2 && r[len(r)-1] == 't' && strings.ContainsRune("aeoö", r[len(r)-2]) {
			p0, link, exception = string(r[:len(r)-2]), string(r[len(r)-2]), true
			h, _, _ = HarmonyOf(key)
			steps = append(steps, key+" is een uitzondering: stam "+p0+"-, verbindingsklinker -"+link+"- (net als "+acc+")")
		}
	}
	if p0 == "" && isVowel(lastRune(w)) {
		stem, step := lengthen(w)
		if step != "" {
			steps = append(steps, step)
		}
		ending := [6]string{"m", "d", two("ja", "je"), "nk", pick(h, "tok", "tek", "tök"), two("juk", "jük")}[p]
		return Form{Text: stem + ending, Steps: append(steps, "eindigt op een klinker: -"+ending)}, true
	}
	if p0 == "" {
		if _, _, regular := lookupSuffix(regularNouns, w); riskyStem(w) && !regular {
			return Form{}, false // korte woorden veranderen vaak van stam: niet zeker
		}
		p0, link = w, pick(h, "o", "e", "ö")
	}
	ti := map[string]string{"a": "tok", "o": "tok", "e": "tek", "ö": "tök"}[link]
	switch p {
	case Sg1, Sg2:
		e := link + map[Person]string{Sg1: "m", Sg2: "d"}[p]
		return Form{Text: p0 + e, Steps: append(steps, "na een medeklinker met verbindingsklinker: -"+e)}, true
	case Pl1:
		e := two("unk", "ünk")
		return Form{Text: p0 + e, Steps: append(steps, "mi: -"+e)}, true
	case Pl2:
		return Form{Text: p0 + link + ti, Steps: append(steps, "ti: -"+link+ti)}, true
	}
	// ő en ők: met of zonder j. Woorden die van stam veranderen nemen geen j (háza, lova, sarka),
	// behalve wat in de lijst staat (földje).
	j, jwhy, certain := possTakesJ(w)
	if _, _, listed := lookupSuffix(possJ, w); exception && !listed {
		j, jwhy, certain = false, "stam verandert: -a/-e zonder j", true
	}
	if !certain {
		return Form{}, false // háza of kertje: niet zeker, niet vragen
	}
	steps = append(steps, jwhy)
	if p == Sg3 {
		e := two("a", "e")
		if j {
			e = two("ja", "je")
		}
		return Form{Text: p0 + e, Steps: append(steps, "ő: -"+e)}, true
	}
	e := two("uk", "ük")
	if j {
		e = two("juk", "jük")
	}
	return Form{Text: p0 + e, Steps: append(steps, "ők: -"+e)}, true
}
