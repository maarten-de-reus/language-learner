// Gebiedende wijs (felszólító mód): tanulj!, olvass!, nézz! Ook de aanvoegende wijs (hogy
// olvassak). De j van de uitgang smelt samen met de laatste medeklinker van de stam.

package grammar

import "strings"

// impIrregular: onbepaald (én … ők) en, bij werkwoorden met een lijdend voorwerp, bepaald
var impIrregular = map[string]struct {
	forms, definite [6]string
	also, alsoDef   map[int]string // ook goed, per persoon (jöjj naast gyere, egyed naast edd)
	note            string
}{
	"van":     {forms: [6]string{"legyek", "legyél", "legyen", "legyünk", "legyetek", "legyenek"}, also: map[int]string{1: "légy"}, note: "van/lesz: legyen"},
	"lesz":    {forms: [6]string{"legyek", "legyél", "legyen", "legyünk", "legyetek", "legyenek"}, also: map[int]string{1: "légy"}, note: "van/lesz: legyen"},
	"megy":    {forms: [6]string{"menjek", "menj", "menjen", "menjünk", "menjetek", "menjenek"}, also: map[int]string{1: "menjél"}, note: "megy: menj"},
	"jön":     {forms: [6]string{"jöjjek", "gyere", "jöjjön", "jöjjünk", "gyertek", "jöjjenek"}, also: map[int]string{1: "jöjj", 4: "jöjjetek"}, note: "jön: gyere, gyertek (of jöjj, jöjjetek)"},
	"eszik":   {forms: [6]string{"egyek", "egyél", "egyen", "együnk", "egyetek", "egyenek"}, definite: [6]string{"egyem", "edd", "egye", "együk", "egyétek", "egyék"}, also: map[int]string{1: "egyed"}, alsoDef: map[int]string{1: "egyed"}, note: "eszik: egyél, edd"},
	"iszik":   {forms: [6]string{"igyak", "igyál", "igyon", "igyunk", "igyatok", "igyanak"}, definite: [6]string{"igyam", "idd", "igya", "igyuk", "igyátok", "igyák"}, also: map[int]string{1: "igyad"}, alsoDef: map[int]string{1: "igyad"}, note: "iszik: igyál, idd"},
	"tesz":    {forms: [6]string{"tegyek", "tegyél", "tegyen", "tegyünk", "tegyetek", "tegyenek"}, definite: [6]string{"tegyem", "tedd", "tegye", "tegyük", "tegyétek", "tegyék"}, also: map[int]string{1: "tégy"}, alsoDef: map[int]string{1: "tegyed"}, note: "tesz: tegyél, tedd"},
	"vesz":    {forms: [6]string{"vegyek", "vegyél", "vegyen", "vegyünk", "vegyetek", "vegyenek"}, definite: [6]string{"vegyem", "vedd", "vegye", "vegyük", "vegyétek", "vegyék"}, also: map[int]string{1: "végy"}, alsoDef: map[int]string{1: "vegyed"}, note: "vesz: vegyél, vedd"},
	"visz":    {forms: [6]string{"vigyek", "vigyél", "vigyen", "vigyünk", "vigyetek", "vigyenek"}, definite: [6]string{"vigyem", "vidd", "vigye", "vigyük", "vigyétek", "vigyék"}, also: map[int]string{1: "vígy"}, alsoDef: map[int]string{1: "vigyed"}, note: "visz: vigyél, vidd"},
	"hisz":    {forms: [6]string{"higgyek", "higgyél", "higgyen", "higgyünk", "higgyetek", "higgyenek"}, definite: [6]string{"higgyem", "hidd", "higgye", "higgyük", "higgyétek", "higgyék"}, also: map[int]string{1: "higgy"}, alsoDef: map[int]string{1: "higgyed"}, note: "hisz: higgyél, hidd"},
	"alszik":  {forms: [6]string{"aludjak", "aludj", "aludjon", "aludjunk", "aludjatok", "aludjanak"}, also: map[int]string{1: "aludjál"}, note: "alszik: aludj (stam alud-)"},
	"fürdik":  {forms: [6]string{"fürödjek", "fürödj", "fürödjön", "fürödjünk", "fürödjetek", "fürödjenek"}, also: map[int]string{1: "fürödjél"}, note: "fürdik: fürödj (stam füröd-)"},
	"fekszik": {forms: [6]string{"feküdjek", "feküdj", "feküdjön", "feküdjünk", "feküdjetek", "feküdjenek"}, also: map[int]string{1: "feküdjél"}, note: "fekszik: feküdj (stam feküd-)"},
}

// impStem geeft de stam met de j erin (tanulj-, olvass-, láss-, taníts-) en de korte bepaalde
// jij-vorm (tanuld, olvasd, lásd, tanítsd)
func impStem(stem string) (s, shortDef, step string, ok bool) {
	cons, double := finalConsonant(stem)
	if cons == "" {
		return "", "", "", false // eindigt op een klinker: niet zeker
	}
	before := strings.TrimSuffix(stem, cons)
	switch {
	case cons == "s" || cons == "sz" || cons == "z" || cons == "dz" || cons == "zs":
		if double {
			return stem, stem + "d", "eindigt al op een dubbele " + cons + ": de j wordt daarin opgenomen", true
		}
		return doubleFinal(stem, cons), stem + "d", "na " + cons + " wordt de j ook " + cons + " (" + stem + " → " + doubleFinal(stem, cons) + ")", true
	case cons == "t" && strings.HasSuffix(stem, "st"):
		b := strings.TrimSuffix(stem, "st")
		return b + "ss", b + "sd", "st + j wordt ss (" + stem + " → " + b + "ss)", true
	case cons == "t" && strings.HasSuffix(stem, "szt"):
		b := strings.TrimSuffix(stem, "szt")
		return b + "ssz", b + "szd", "szt + j wordt ssz (" + stem + " → " + b + "ssz)", true
	case cons == "t" && (strings.HasSuffix(stem, "ít") || (before != "" && !isVowel(lastRune(before)))):
		return stem + "s", stem + "sd", "na -ít of medeklinker + t wordt de j een s: ts (" + stem + " → " + stem + "s)", true
	case cons == "t" && !double:
		b := strings.TrimSuffix(stem, "t")
		return b + "ss", b + "sd", "klinker + t + j wordt ss (" + stem + " → " + b + "ss)", true
	case double && cons != "l":
		return "", "", "", false // dubbele medeklinker (behalve ll): niet zeker
	}
	return stem + "j", stem + "d", "de j komt gewoon achter de stam (" + stem + "j-)", true
}

// Imperative: gebiedende wijs, onbepaald of bepaald. Bij 'te' is de korte vorm het antwoord
// (olvass, olvasd) en de lange ook goed (olvassál, olvassad).
func Imperative(verb string, p Person, definite bool) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	if verb == "" || strings.ContainsAny(verb, " -") || notConjugated[verb] {
		return Form{}, false
	}
	base := baseVerb(verb)
	prefix := strings.TrimSuffix(verb, base)
	stem, ik := verbStem(base)
	if definite && (intransitive[base] || (ik && !transitiveIk[base])) {
		return Form{}, false
	}
	if irr, ok := impIrregular[base]; ok {
		table := irr.forms
		if definite {
			table = irr.definite
		}
		if table[p] == "" {
			return Form{}, false
		}
		f := Form{Text: prefix + table[p], Steps: []string{irr.note}}
		also := irr.also
		if definite {
			also = irr.alsoDef
		}
		if a, ok := also[int(p)]; ok {
			f.Also = []string{prefix + a}
		}
		return f, true
	}
	s, shortDef, step, ok := impStem(stem)
	if !ok {
		return Form{}, false
	}
	h, _, why := HarmonyOf(stem) // de stam zonder -ik: fejlőd -> fejlődjön
	back := h == Back
	steps := []string{"stam " + stem + ": " + why, step}
	form := func(ending string, also ...string) (Form, bool) {
		f := Form{Text: prefix + s + ending, Steps: append(steps, PersonLabels[p]+": -"+ending+" achter "+s)}
		for _, a := range also {
			f.Also = append(f.Also, prefix+a)
		}
		return f, true
	}
	pick2 := func(b, f string) string {
		if back {
			return b
		}
		return f
	}
	if definite {
		switch p {
		case Sg1:
			return form(pick2("am", "em"))
		case Sg2:
			f := Form{Text: prefix + shortDef, Steps: append(steps, "te, bepaald: kort met -d ("+shortDef+"), lang met -"+pick2("ad", "ed")+" ("+s+pick2("ad", "ed")+")"),
				Also: []string{prefix + s + pick2("ad", "ed")}}
			return f, true
		case Sg3:
			return form(pick2("a", "e"))
		case Pl1:
			return form(pick2("uk", "ük"))
		case Pl2:
			return form(pick2("átok", "étek"))
		default:
			return form(pick2("ák", "ék"))
		}
	}
	switch p {
	case Sg1:
		if ik {
			return form(pick2("ak", "ek"), s+pick2("am", "em")) // -ik: dolgozzak of dolgozzam
		}
		return form(pick2("ak", "ek"))
	case Sg2:
		f := Form{Text: prefix + s, Steps: append(steps, "te: kort, alleen de stam met j ("+s+"), lang -"+pick2("ál", "él")+" ("+s+pick2("ál", "él")+")"),
			Also: []string{prefix + s + pick2("ál", "él")}}
		return f, true
	case Sg3:
		return form(pick(h, "on", "en", "ön"))
	case Pl1:
		return form(pick2("unk", "ünk"))
	case Pl2:
		return form(pick2("atok", "etek"))
	default:
		return form(pick2("anak", "enek"))
	}
}
