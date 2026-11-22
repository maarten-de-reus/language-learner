// Voorwaardelijke wijs (feltételes mód): olvasnék, szeretnék, lennék. Gemaakt uit de infinitief
// zonder -ni (olvas-ni -> olvasnék, taníta-ni -> tanítanék, en-ni -> ennék), dus de onregelmatige
// infinitieven (enni, inni, lenni, menni …) gaan vanzelf goed. De verleden vorm is de verleden
// tijd + volna (olvastam volna).

package grammar

import "strings"

// condStem: de infinitief zonder -ni
func condStem(inf string) (string, bool) {
	if !strings.HasSuffix(inf, "ni") || len(inf) < 4 {
		return "", false
	}
	return strings.TrimSuffix(inf, "ni"), true
}

// Conditional: tegenwoordige voorwaardelijke wijs, onbepaald of bepaald. inf is de infinitief
// (ook een afwijkende infinitief bij het woord).
func Conditional(verb, inf string, p Person, definite bool) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	base := baseVerb(verb)
	_, ik := verbStem(base)
	if definite && (intransitive[base] || (ik && !transitiveIk[base])) {
		return Form{}, false
	}
	stem, ok := condStem(inf)
	if !ok {
		return Form{}, false
	}
	// Klinkerharmonie van de infinitief (lenni -> lennétek), behalve bij een werkwoord dat zelf een
	// uitzondering is (iszik -> inni, maar innál met achterklinkers)
	h, _, why := HarmonyOf(stem)
	vs, _ := verbStem(base)
	if eh, ok := harmonyExceptions[vs]; ok {
		h, why = eh, vs+" is een uitzondering: "+eh.String()
	} else if eh, ok := harmonyExceptions[base]; ok {
		h, why = eh, base+" is een uitzondering: "+eh.String()
	}
	back := h == Back
	var ending string
	if definite {
		ending = [6]string{"nám", "nád", "ná", "nánk", "nátok", "nák"}[p]
		if !back {
			ending = [6]string{"ném", "néd", "né", "nénk", "nétek", "nék"}[p]
		}
	} else {
		ending = [6]string{"nék", "nál", "na", "nánk", "nátok", "nának"}[p]
		if !back {
			ending = [6]string{"nék", "nél", "ne", "nénk", "nétek", "nének"}[p]
		}
	}
	steps := []string{"voorwaardelijke wijs: de infinitief zonder -ni (" + inf + " → " + stem + "-)", "klinkers: " + why,
		PersonLabels[p] + ": -" + ending}
	f := Form{Text: stem + ending, Steps: steps}
	if !definite && p == Sg1 && ik {
		f.Also = []string{stem + map[bool]string{true: "nám", false: "ném"}[back]} // -ik-werkwoorden: dolgoznék of dolgoznám
	}
	if !definite && p == Sg3 && (base == "van" || base == "lesz") {
		f.Also = []string{strings.TrimSuffix(verb, base) + "volna"} // lenne of volna
	}
	return f, true
}

// PastConditional: verleden voorwaardelijke wijs = verleden tijd + volna (olvastam volna)
func PastConditional(verb string, p Person) (Form, bool) {
	past, ok := Past(verb, p, false)
	if !ok {
		return Form{}, false
	}
	return Form{Text: past.Text + " volna", Steps: append(past.Steps, "verleden voorwaardelijk: de verleden tijd ("+past.Text+") + volna")}, true
}
