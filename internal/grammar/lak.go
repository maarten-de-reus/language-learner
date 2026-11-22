// -lak/-lek: ik … jou (látlak, szeretlek). Eén vorm, alleen bij werkwoorden met een lijdend voorwerp.

package grammar

import "strings"

// Lak maakt de -lak/-lek-vorm uit de 3e persoon enkelvoud
func Lak(verb string) (Form, bool) {
	verb = strings.ToLower(strings.TrimSpace(verb))
	if verb == "" || strings.ContainsAny(verb, " -") || notConjugated[verb] || isVowel(lastRune(verb)) {
		return Form{}, false
	}
	base := baseVerb(verb)
	stem, ik := verbStem(verb)
	if intransitive[base] || (ik && !transitiveIk[base]) {
		return Form{}, false // zonder lijdend voorwerp geen 'ik … jou'
	}
	h, _, why := HarmonyOf(stem)
	steps := []string{"stam " + stem + ": " + why, "ik … jou: één vorm, -" + pick(h, "lak", "lek") + " achter de stam"}
	switch {
	case strings.HasSuffix(stem, "ll"):
		e := pick(h, "ak", "ek")
		return Form{Text: stem + e, Steps: append(steps, "eindigt al op ll: geen derde l ("+stem+e+")")}, true
	case needsLink(stem):
		e := pick(h, "alak", "elek")
		return Form{Text: stem + e, Steps: append(steps, "eindigt op -ít of twee medeklinkers: verbindingsklinker, dus -"+e)}, true
	}
	return Form{Text: stem + pick(h, "lak", "lek"), Steps: steps}, true
}
