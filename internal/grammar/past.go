// Verleden tijd, toekomende tijd (fog + infinitief) en -hat/-het (kunnen, mogen), gemaakt uit de
// 3e persoon enkelvoud, net als de tegenwoordige tijd.

package grammar

import "strings"

// pastIrregular: onregelmatige verleden tijden, onbepaald (én … ők) en het verleden-stuk voor de
// bepaalde vervoeging (ett- bij eszik: ettem, etted, ette …)
var pastIrregular = map[string]struct {
	forms [6]string
	stem  string // voor de bepaalde vervoeging; leeg = geen (onovergankelijk)
	note  string
}{
	"van":       {[6]string{"voltam", "voltál", "volt", "voltunk", "voltatok", "voltak"}, "", "van: volt"},
	"lesz":      {[6]string{"lettem", "lettél", "lett", "lettünk", "lettetek", "lettek"}, "", "lesz: lett"},
	"megy":      {[6]string{"mentem", "mentél", "ment", "mentünk", "mentetek", "mentek"}, "", "megy: ment"},
	"jön":       {[6]string{"jöttem", "jöttél", "jött", "jöttünk", "jöttetek", "jöttek"}, "", "jön: jött"},
	"eszik":     {[6]string{"ettem", "ettél", "evett", "ettünk", "ettetek", "ettek"}, "ett", "eszik: ettem, maar ő evett"},
	"iszik":     {[6]string{"ittam", "ittál", "ivott", "ittunk", "ittatok", "ittak"}, "itt", "iszik: ittam, maar ő ivott"},
	"alszik":    {[6]string{"aludtam", "aludtál", "aludt", "aludtunk", "aludtatok", "aludtak"}, "", "alszik: aludt (stam alud-)"},
	"fekszik":   {[6]string{"feküdtem", "feküdtél", "feküdt", "feküdtünk", "feküdtetek", "feküdtek"}, "", "fekszik: feküdt (stam feküd-)"},
	"haragszik": {[6]string{"haragudtam", "haragudtál", "haragudott", "haragudtunk", "haragudtatok", "haragudtak"}, "", "haragszik: haragudott"},
	"tesz":      {[6]string{"tettem", "tettél", "tett", "tettünk", "tettetek", "tettek"}, "tett", "tesz: tett"},
	"vesz":      {[6]string{"vettem", "vettél", "vett", "vettünk", "vettetek", "vettek"}, "vett", "vesz: vett"},
	"visz":      {[6]string{"vittem", "vittél", "vitt", "vittünk", "vittetek", "vittek"}, "vitt", "visz: vitt"},
	"hisz":      {[6]string{"hittem", "hittél", "hitt", "hittünk", "hittetek", "hittek"}, "hitt", "hisz: hitt"},
	"lő":        {[6]string{"lőttem", "lőttél", "lőtt", "lőttünk", "lőttetek", "lőttek"}, "lőtt", "lő: lőtt"},
	"nő":        {[6]string{"nőttem", "nőttél", "nőtt", "nőttünk", "nőttetek", "nőttek"}, "", "nő: nőtt"},
	"fürdik":    {[6]string{"fürödtem", "fürödtél", "fürdött", "fürödtünk", "fürödtetek", "fürödtek"}, "", "fürdik: fürödtem, fürdött"},
	"hall":      {[6]string{"hallottam", "hallottál", "hallott", "hallottunk", "hallottatok", "hallottak"}, "hallott", "hall: hallott"},
	"áll":       {[6]string{"álltam", "álltál", "állt", "álltunk", "álltatok", "álltak"}, "", "áll: állt"},
	"száll":     {[6]string{"szálltam", "szálltál", "szállt", "szálltunk", "szálltatok", "szálltak"}, "", "száll: szállt"},
	"lát":       {[6]string{"láttam", "láttál", "látott", "láttunk", "láttatok", "láttak"}, "látt", "lát: láttam, maar ő látott"},
	"mosolyog":  {[6]string{"mosolyogtam", "mosolyogtál", "mosolygott", "mosolyogtunk", "mosolyogtatok", "mosolyogtak"}, "", "mosolyog: ő mosolygott (de o valt weg)"},
	"marad":     {[6]string{"maradtam", "maradtál", "maradt", "maradtunk", "maradtatok", "maradtak"}, "", "marad: maradt"},
}

// directPast: na deze medeklinkers (één, niet dubbel) krijgt ook ő geen verbindingsklinker: tanult,
// beszélt, várt, kért
var directPast = []string{"j", "l", "ly", "n", "ny", "r"}

// pastParts geeft het verleden-stuk voor én … ők (behalve ő) en de vorm voor ő
func pastParts(verb string) (stem, third string, steps []string, ok bool) {
	st, ik := verbStem(verb)
	h, _, why := HarmonyOf(st)
	link := pick(h, "ott", "ett", "ött")
	steps = []string{"stam " + st + ": " + why}
	cons, double := finalConsonant(st)
	if cons == "" || double {
		return "", "", nil, false // klinker of dubbele medeklinker aan het eind: zie de lijst
	}
	before := strings.TrimSuffix(st, cons)
	// Medeklinker + t of + s/sz/z (ért, tart, fest, játsz): overal een verbindingsklinker
	clusterT := (cons == "t" || cons == "s" || cons == "sz" || cons == "z") && before != "" && !isVowel(lastRune(before))
	shortT := cons == "t" && !clusterT && syllables(st) == 1

	// De andere personen: verbindingsklinker na -ít, medeklinker + t en korte stammen op -t (futottam)
	if strings.HasSuffix(st, "ít") || clusterT || shortT {
		stem = st + link
		steps = append(steps, "eindigt op -ít, op medeklinker + t/s/sz/z of op -t in één lettergreep: -"+link+"- in alle personen")
	} else {
		stem = st + "t"
		steps = append(steps, "verleden tijd: -t- achter de stam ("+stem+"-)")
	}
	// ő
	switch {
	case stem != st+"t":
		third = stem
	case contains(directPast, cons):
		third = st + "t"
		steps = append(steps, "na j, l, ly, n, ny, r krijgt ő alleen -t ("+third+")")
	case cons == "d" && syllables(st) > 1 && !ik:
		return "", "", nil, false // marad -> maradt, maar enged -> engedett: niet zeker
	default:
		third = st + link
		steps = append(steps, "ő krijgt een verbindingsklinker: -"+link+" ("+third+")")
	}
	return stem, third, steps, true
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Past: verleden tijd, onbepaald of bepaald
func Past(verb string, p Person, definite bool) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	if verb == "" || strings.ContainsAny(verb, " -") || notConjugated[verb] {
		return Form{}, false
	}
	base := baseVerb(verb)
	_, ik := verbStem(verb)
	if definite && (intransitive[base] || (ik && !transitiveIk[base])) {
		return Form{}, false
	}
	prefix := ""
	irr, isIrr := pastIrregular[verb]
	if !isIrr {
		for _, px := range prefixes {
			if rest := strings.TrimPrefix(verb, px); rest != verb {
				if irr, isIrr = pastIrregular[rest]; isIrr {
					prefix = px
					break
				}
			}
		}
	}
	var stem, third string
	var steps []string
	if isIrr {
		if !definite {
			return Form{Text: prefix + irr.forms[p], Steps: []string{irr.note}}, true
		}
		if irr.stem == "" {
			return Form{}, false
		}
		stem, steps = prefix+irr.stem, []string{irr.note}
	} else {
		var ok bool
		if stem, third, steps, ok = pastParts(verb); !ok {
			return Form{}, false
		}
	}
	// Klinkerharmonie van het werkwoord zelf, niet van het verleden-stuk (ír -> írtam, iszik -> ittam)
	vs, _ := verbStem(baseVerb(verb))
	h, _, _ := HarmonyOf(vs)
	back := h == Back
	var ending string
	if definite {
		ending = [6]string{"am", "ad", "a", "uk", "átok", "ák"}[p]
		if !back {
			ending = [6]string{"em", "ed", "e", "ük", "étek", "ék"}[p]
		}
		return Form{Text: stem + ending, Steps: append(steps, "bepaald, "+PersonLabels[p]+": -"+ending)}, true
	}
	if p == Sg3 {
		return Form{Text: third, Steps: steps}, true
	}
	ending = [6]string{"am", "ál", "", "unk", "atok", "ak"}[p]
	if !back {
		ending = [6]string{"em", "él", "", "ünk", "etek", "ek"}[p]
	}
	return Form{Text: stem + ending, Steps: append(steps, PersonLabels[p]+": -"+ending)}, true
}

// ---------- fog + infinitief ----------

var fogForms = [6]string{"fogok", "fogsz", "fog", "fogunk", "fogtok", "fognak"}

// Future: fog (vervoegd) + infinitief. Beide volgordes zijn goed: olvasni fogok / fogok olvasni.
func Future(inf string, p Person) (Form, bool) {
	if inf == "" {
		return Form{}, false
	}
	return Form{Text: inf + " " + fogForms[p], Also: []string{fogForms[p] + " " + inf}, Steps: []string{
		"toekomende tijd: de infinitief (" + inf + ") met fog, vervoegd als gewoon werkwoord: fogok, fogsz, fog, fogunk, fogtok, fognak",
		"de volgorde mag allebei: " + inf + " " + fogForms[p] + " of " + fogForms[p] + " " + inf,
	}}, true
}

// ---------- -hat/-het ----------

var hatIrregular = map[string]string{
	"van": "lehet", "lesz": "lehet", "megy": "mehet", "jön": "jöhet", "eszik": "ehet", "iszik": "ihat",
	"tesz": "tehet", "vesz": "vehet", "visz": "vihet", "hisz": "hihet", "alszik": "alhat", "fekszik": "fekhet",
}

// Potential: -hat/-het (kunnen, mogen), vervoegd als een gewoon werkwoord: olvashatok, olvashatsz …
func Potential(verb string, p Person) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	if verb == "" || strings.ContainsAny(verb, " -") || notConjugated[verb] {
		return Form{}, false
	}
	base := baseVerb(verb)
	prefix := strings.TrimSuffix(verb, base)
	var hat string
	var steps []string
	if irr, ok := hatIrregular[base]; ok {
		hat = prefix + irr
		steps = []string{base + " is onregelmatig: " + irr}
	} else {
		st, _ := verbStem(verb)
		h, _, why := HarmonyOf(st)
		hat = st + pick(h, "hat", "het")
		steps = []string{"stam " + st + ": " + why, "-" + pick(h, "hat", "het") + " achter de stam: " + hat}
	}
	return conjugateHat(hat, p, steps), true
}

// conjugateHat vervoegt een -hat/-het-vorm als gewoon werkwoord (ook voor een afwijkende vorm)
func conjugateHat(hat string, p Person, steps []string) Form {
	back := strings.HasSuffix(hat, "hat")
	ending := [6]string{"ok", "sz", "", "unk", "tok", "nak"}[p]
	if !back {
		ending = [6]string{"ek", "sz", "", "ünk", "tek", "nek"}[p]
	}
	if p != Sg3 {
		steps = append(steps, "vervoegd als gewoon werkwoord: "+PersonLabels[p]+" -"+ending)
	}
	return Form{Text: hat + ending, Steps: steps}
}
