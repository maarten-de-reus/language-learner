// Werkwoorden: de tegenwoordige tijd (onbepaald en bepaald) en de infinitief, gemaakt uit de
// 3e persoon enkelvoud (olvas, lakik). Dat is de vorm die in het woordenboek staat en die je leert.

package grammar

import (
	"strings"
)

// Person: én, te, ő, mi, ti, ők
type Person int

const (
	Sg1 Person = iota
	Sg2
	Sg3
	Pl1
	Pl2
	Pl3
)

// PersonLabels: zo staan de personen in de oefening
var PersonLabels = [6]string{"én (ik)", "te (jij)", "ő (hij/zij)", "mi (wij)", "ti (jullie)", "ők (zij)"}

// Onregelmatige werkwoorden: tegenwoordige tijd onbepaald, en de infinitief
type irregularVerb struct {
	present    [6]string // leeg: deze vorm kennen de regels niet zeker
	definite   [6]string // leeg = geen bepaalde vervoeging (onovergankelijk) of volgt de regel
	infinitive string
	note       string
}

var irregularVerbs = map[string]irregularVerb{
	"van":       {present: [6]string{"vagyok", "vagy", "van", "vagyunk", "vagytok", "vannak"}, infinitive: "lenni", note: "van is helemaal onregelmatig"},
	"lesz":      {present: [6]string{"leszek", "leszel", "lesz", "leszünk", "lesztek", "lesznek"}, infinitive: "lenni", note: "lesz (worden) heeft dezelfde infinitief als van: lenni"},
	"megy":      {present: [6]string{"megyek", "mész", "megy", "megyünk", "mentek", "mennek"}, infinitive: "menni", note: "megy is onregelmatig: mész, mentek, mennek"},
	"jön":       {present: [6]string{"jövök", "jössz", "jön", "jövünk", "jöttök", "jönnek"}, infinitive: "jönni", note: "jön is onregelmatig: jövök, jössz"},
	"eszik":     {infinitive: "enni", note: "eszik/iszik: de vervoeging is regelmatig, de infinitief niet (enni)"},
	"iszik":     {infinitive: "inni", note: "eszik/iszik: de vervoeging is regelmatig, de infinitief niet (inni)"},
	"alszik":    {present: [6]string{"alszom", "alszol", "alszik", "alszunk", "alszotok", "alszanak"}, infinitive: "aludni", note: "alszik vervoegt regelmatig (alszom, alszol, alszotok), maar de infinitief is aludni"},
	"fekszik":   {present: [6]string{"fekszem", "fekszel", "fekszik", "fekszünk", "feküsztök", "feküsznek"}, infinitive: "feküdni", note: "fekszik: feküsztök, feküsznek, feküdni"},
	"haragszik": {present: [6]string{"haragszom", "haragszol", "haragszik", "haragszunk", "haragszotok", "haragszanak"}, infinitive: "haragudni", note: "haragszik vervoegt regelmatig, maar de infinitief is haragudni"},
	"tesz":      {infinitive: "tenni", note: "tesz/vesz/visz/hisz: vervoeging regelmatig, infinitief met -nni"},
	"vesz":      {infinitive: "venni", note: "tesz/vesz/visz/hisz: vervoeging regelmatig, infinitief met -nni"},
	"visz":      {infinitive: "vinni", note: "tesz/vesz/visz/hisz: vervoeging regelmatig, infinitief met -nni"},
	"hisz":      {infinitive: "hinni", note: "tesz/vesz/visz/hisz: vervoeging regelmatig, infinitief met -nni"},
	"fürdik":    {present: [6]string{"fürdök", "fürdesz", "fürdik", "fürdünk", "fürödtök", "fürdenek"}, infinitive: "fürdeni", note: "fürdik: fürödtök"},
	"hall":      {present: [6]string{"hallok", "hallasz", "hall", "hallunk", "halltok", "hallanak"}, infinitive: "hallani", note: "hall: hallasz, hallanak, hallani (met verbindingsklinker)"},
	"áll":       {present: [6]string{"állok", "állsz", "áll", "állunk", "álltok", "állnak"}, infinitive: "állni", note: "áll: zonder verbindingsklinker"},
	"száll":     {present: [6]string{"szállok", "szállsz", "száll", "szállunk", "szálltok", "szállnak"}, infinitive: "szállni", note: "száll: zonder verbindingsklinker"},
}

// Werkwoorden die geen lijdend voorwerp nemen: daar bestaat geen bepaalde vervoeging.
// -ik-werkwoorden zijn dat meestal ook; eszik, iszik en játszik niet.
var intransitive = map[string]bool{
	"van": true, "lesz": true, "megy": true, "jön": true, "él": true, "marad": true, "sétál": true,
	"táncol": true, "fordul": true, "jár": true, "nyaral": true, "pihen": true, "ebédel": true,
	"vacsorázik": true, "reggelizik": true, "koncentrál": true, "dönt": true, "tankol": true,
	"áll": true, "száll": true, "fut": true, "ül": true, "siet": true, "lehet": true, "kell": true,
	"mosolyog": true, "örül": true, "repül": true, "kiabál": true, "telefonál": true, "sportol": true,
	"sír": true, "nevet": true, "fél": true, "ér": true,
}
var transitiveIk = map[string]bool{"eszik": true, "iszik": true, "játszik": true}

// Woorden die als werkwoord in je lijst staan maar niet te vervoegen zijn
var notConjugated = map[string]bool{"nincs": true, "lehet": true, "kell": true, "szabad": true, "tilos": true, "szokott": true}

// Voorvoegsels: elmegy vervoegt als megy, met el- ervoor
var prefixes = []string{"vissza", "tovább", "össze", "végig", "haza", "újra", "szét", "meg", "fel", "föl", "oda", "ide", "át", "el", "le", "ki", "be", "rá"}

// splitIrregular zoekt een onregelmatig werkwoord, ook met voorvoegsel (elmegy = el + megy)
func splitIrregular(verb string) (prefix string, iv irregularVerb, ok bool) {
	if iv, ok := irregularVerbs[verb]; ok {
		return "", iv, true
	}
	for _, p := range prefixes {
		if rest := strings.TrimPrefix(verb, p); rest != verb {
			if iv, ok := irregularVerbs[rest]; ok {
				return p, iv, true
			}
		}
	}
	return "", irregularVerb{}, false
}

func baseVerb(verb string) string {
	if p, _, ok := splitIrregular(verb); ok {
		return strings.TrimPrefix(verb, p)
	}
	return verb
}

// verbStem: de stam zonder -ik (lakik -> lak)
func verbStem(verb string) (stem string, ik bool) {
	if strings.HasSuffix(verb, "ik") && len(verb) > 3 {
		return strings.TrimSuffix(verb, "ik"), true
	}
	return verb, false
}

func sibilant(stem string) bool {
	c, _ := finalConsonant(stem)
	return c == "s" || c == "sz" || c == "z" || c == "dz"
}

// needsLink: -ít of twee medeklinkers aan het eind (tanít, mond, ért): dan komt er een
// verbindingsklinker bij -sz, -nak/-nek en de infinitief (tanítasz, mondanak, érteni)
func needsLink(stem string) bool {
	return strings.HasSuffix(stem, "ít") || consonantCluster(stem)
}

// clusterSzik: -szik met een andere medeklinker voor de sz (alszik, fekszik, haragszik), behalve
// játszik: die veranderen vaak van stam
func clusterSzik(stem string, ik bool) bool {
	if !ik || !strings.HasSuffix(stem, "sz") {
		return false
	}
	before := strings.TrimSuffix(stem, "sz")
	return before != "" && !isVowel(lastRune(before)) && !strings.HasSuffix(before, "t")
}

// Present geeft de tegenwoordige tijd (onbepaald of bepaald) voor een persoon. ok is false als de
// regels de vorm niet zeker weten (of als er geen bepaalde vervoeging bestaat).
func Present(verb string, p Person, definite bool) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	if verb == "" || strings.ContainsAny(verb, " -") || notConjugated[verb] || isVowel(lastRune(verb)) {
		return Form{}, false
	}
	base := baseVerb(verb)
	stem, ik := verbStem(verb)
	if definite && (intransitive[base] || (ik && !transitiveIk[base])) {
		return Form{}, false
	}
	if prefix, iv, ok := splitIrregular(verb); ok {
		table := iv.present
		if definite {
			table = iv.definite
		}
		if table[p] != "" {
			return Form{Text: prefix + table[p], Steps: []string{base + " is onregelmatig", iv.note}}, true
		}
		if !definite && iv.present != [6]string{} {
			return Form{}, false
		}
	}
	if p == Sg3 && !definite {
		return Form{Text: verb, Steps: []string{"de 3e persoon enkelvoud is het woord zelf: " + verb}}, true
	}

	h, _, why := HarmonyOf(stem)
	steps := []string{"stam " + stem + func() string {
		if ik {
			return " (zonder -ik)"
		}
		return ""
	}() + ": " + why}
	if _, double := finalConsonant(stem); double && (p == Sg2 || p == Pl3) && !definite {
		return Form{}, false // hall -> hallasz, maar áll -> állsz: niet zeker
	}
	if clusterSzik(stem, ik) && (p == Pl2 || p == Pl3) && !definite {
		return Form{}, false // fekszik -> feküsztök: niet zeker
	}
	add := func(ending, step string) (Form, bool) {
		return Form{Text: stem + ending, Steps: append(steps, step)}, true
	}
	if definite {
		return definiteForm(verb, stem, h, p, steps)
	}
	switch p {
	case Sg1:
		if ik {
			e := pick(h, "om", "em", "öm")
			return add(e, "-ik-werkwoord: ik-vorm op -m, dus -"+e)
		}
		e := pick(h, "ok", "ek", "ök")
		return add(e, "ik-vorm: -"+e)
	case Sg2:
		if sibilant(stem) {
			e := pick(h, "ol", "el", "öl")
			return add(e, "eindigt op s, sz of z: geen -sz maar -"+e)
		}
		if needsLink(stem) {
			e := pick(h, "asz", "esz")
			return add(e, "eindigt op -ít of twee medeklinkers: verbindingsklinker, dus -"+e)
		}
		return add("sz", "jij-vorm: -sz")
	case Sg3:
		return Form{Text: verb, Steps: steps}, true
	case Pl1:
		e := pick(h, "unk", "ünk")
		return add(e, "wij-vorm: -"+e)
	case Pl2:
		last, _ := finalConsonant(stem)
		if strings.HasSuffix(stem, "ít") || (consonantCluster(stem) && (last == "t" || sibilant(stem))) {
			e := pick(h, "otok", "etek", "ötök")
			return add(e, "eindigt op -ít of medeklinker + t/s/sz/z: verbindingsklinker, dus -"+e)
		}
		if last == "t" {
			return Form{}, false // lát -> láttok, maar süt -> sütötök: niet zeker
		}
		e := pick(h, "tok", "tek", "tök")
		return add(e, "jullie-vorm: -"+e)
	case Pl3:
		if needsLink(stem) {
			e := pick(h, "anak", "enek")
			return add(e, "eindigt op -ít of twee medeklinkers: verbindingsklinker, dus -"+e)
		}
		e := pick(h, "nak", "nek")
		return add(e, "zij-vorm (meervoud): -"+e)
	}
	return Form{}, false
}

// definiteForm: bepaalde vervoeging (látom, látod, látja). Na s, sz, z wordt de j gelijk aan die
// medeklinker: olvas + ja -> olvassa, néz + jük -> nézzük.
func definiteForm(verb, stem string, h Harmony, p Person, steps []string) (Form, bool) {
	back := h == Back
	var ending string
	switch p {
	case Sg1:
		ending = pick(h, "om", "em", "öm")
	case Sg2:
		ending = pick(h, "od", "ed", "öd")
	case Sg3:
		ending = map[bool]string{true: "ja", false: "i"}[back]
	case Pl1:
		ending = map[bool]string{true: "juk", false: "jük"}[back]
	case Pl2:
		ending = map[bool]string{true: "játok", false: "itek"}[back]
	case Pl3:
		ending = map[bool]string{true: "ják", false: "ik"}[back]
	}
	steps = append(steps, "bepaalde vervoeging, "+PersonLabels[p]+": -"+ending)
	if strings.HasPrefix(ending, "j") && sibilant(stem) {
		cons, _ := finalConsonant(stem)
		steps = append(steps, "na "+cons+" wordt de j gelijk aan de "+cons+" ("+cons+"+j → "+cons[:1]+cons+")")
		return Form{Text: doubleFinal(stem, cons) + strings.TrimPrefix(ending, "j"), Steps: steps}, true
	}
	return Form{Text: stem + ending, Steps: steps}, true
}

// Infinitive maakt de infinitief uit de 3e persoon enkelvoud: stam + -ni, met verbindingsklinker
// na -ít of twee medeklinkers (tanítani, mondani)
func Infinitive(verb string) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	if verb == "" || strings.ContainsAny(verb, " -") || notConjugated[verb] || isVowel(lastRune(verb)) {
		return Form{}, false
	}
	if prefix, iv, ok := splitIrregular(verb); ok && iv.infinitive != "" {
		return Form{Text: prefix + iv.infinitive, Steps: []string{baseVerb(verb) + " is onregelmatig", iv.note}}, true
	}
	stem, ik := verbStem(verb)
	steps := []string{}
	if ik {
		steps = append(steps, "-ik eraf: "+stem)
	}
	if clusterSzik(stem, ik) {
		return Form{}, false // alszik -> aludni: niet zeker
	}
	if _, double := finalConsonant(stem); double {
		return Form{}, false // hall -> hallani, maar áll -> állni: niet zeker
	}
	if needsLink(stem) {
		h, _, why := HarmonyOf(stem)
		e := pick(h, "ani", "eni")
		return Form{Text: stem + e, Steps: append(steps, why, "eindigt op -ít of twee medeklinkers: verbindingsklinker, dus -"+e)}, true
	}
	return Form{Text: stem + "ni", Steps: append(steps, "-ni erachter (de -ni neemt geen klinkerharmonie)")}, true
}
