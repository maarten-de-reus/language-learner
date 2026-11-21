// Package grammar maakt Hongaarse woordvormen met regels in plaats van uit het hoofd: klinker-
// harmonie, naamvalsuitgangen, meervoud, lijdend voorwerp en de vervoeging van werkwoorden. Elke
// vorm komt met de stappen waarmee hij gemaakt is, zodat de oefening uitlegt waarom. Wat de regels
// niet zeker weten (bijv. een woord dat zijn stam verandert en niet in de uitzonderingen staat)
// geeft geen vorm: liever niet oefenen dan iets fout aanleren.
package grammar

import (
	"strings"
	"unicode/utf8"
)

// Harmony is de klinkergroep die bepaalt welke variant van een uitgang een woord krijgt
type Harmony int

const (
	Back    Harmony = iota // achterklinkers a á o ó u ú: -ban, -hoz
	Front                  // voorklinkers e é i í: -ben, -hez
	Rounded                // voorklinkers met ronde lippen ö ő ü ű: -ben, -höz
)

func (h Harmony) String() string {
	return [...]string{"achterklinkers", "voorklinkers", "ronde voorklinkers"}[h]
}

const (
	backVowels    = "aáoóuú"
	roundedVowels = "öőüű"
	neutralVowels = "iíé" // tellen niet mee na een achterklinker: papír en kávé zijn achter
	vowels        = "aáeéiíoóöőuúüű"
)

func isVowel(r rune) bool { return strings.ContainsRune(vowels, r) }

// Woorden die anders gaan dan hun klinkers doen vermoeden
var harmonyExceptions = map[string]Harmony{
	// i/í-woorden met achterklinker-uitgangen
	"híd": Back, "íj": Back, "nyíl": Back, "zsír": Back, "ír": Back, "nyit": Back, "szív": Back,
	"hív": Back, "sír": Back, "nyír": Back, "iszik": Back, "isz": Back,
}

// Samenstellingen volgen het laatste deel: számítógép -> számítógépben (gép), niet -ban
var frontFinalParts = []string{"gép", "kép", "szék", "rész", "nép", "hír", "év"}

// HarmonyOf geeft de klinkergroep van een woord, en mixed als het woord achterklinkers heeft
// gevolgd door een e (hotel, fotel): dan zijn beide varianten in gebruik.
//
// De regel: na de laatste achterklinker alleen nog i, í of é (of niets): achter. Anders voor, en
// rond als de laatste klinker ö, ő, ü of ű is.
func HarmonyOf(word string) (h Harmony, mixed bool, why string) {
	w := strings.ToLower(lastWord(word))
	if h, ok := harmonyExceptions[w]; ok {
		return h, false, w + " is een uitzondering: " + h.String()
	}
	for _, part := range frontFinalParts {
		if strings.HasSuffix(w, part) && w != part && hasAny(strings.TrimSuffix(w, part), backVowels) {
			return Front, false, "samenstelling: het laatste deel (" + part + ") beslist"
		}
	}
	var seq []rune
	for _, r := range w {
		if isVowel(r) {
			seq = append(seq, r)
		}
	}
	if len(seq) == 0 {
		return Front, false, "geen klinkers"
	}
	lastBack := -1
	for i, r := range seq {
		if strings.ContainsRune(backVowels, r) {
			lastBack = i
		}
	}
	if lastBack >= 0 {
		rest := string(seq[lastBack+1:])
		if strings.Trim(rest, neutralVowels) == "" {
			why := "laatste klinker " + string(seq[len(seq)-1]) + " is een achterklinker"
			if rest != "" {
				why = "achterklinker " + string(seq[lastBack]) + " met daarna alleen i/í/é: achter"
			}
			if len([]rune(rest)) >= 2 {
				return Back, true, "achterklinker " + string(seq[lastBack]) + " met daarna meer i/í/é: allebei in gebruik"
			}
			return Back, false, why
		}
		if strings.ContainsRune(rest, 'e') && !hasAny(rest, roundedVowels) {
			return Front, true, "achterklinker " + string(seq[lastBack]) + " gevolgd door e: allebei in gebruik, meestal voor"
		}
	}
	last := seq[len(seq)-1]
	if strings.ContainsRune(roundedVowels, last) {
		return Rounded, false, "laatste klinker " + string(last) + " is een ronde voorklinker"
	}
	if lastBack < 0 && strings.ContainsRune(neutralVowels, last) && onlyNeutral(seq) {
		return Front, false, "alleen i/í/é: voor"
	}
	// Laatste klinker e/é/i/í, met een ronde klinker ervoor (kávéfőző eindigt op ő en is dus rond)
	return Front, false, "laatste klinker " + string(last) + " is een voorklinker"
}

func onlyNeutral(seq []rune) bool {
	for _, r := range seq {
		if !strings.ContainsRune(neutralVowels, r) {
			return false
		}
	}
	return true
}

func hasAny(s, set string) bool { return strings.ContainsAny(s, set) }

// lastWord: bij een woordgroep telt het laatste woord (jó reggelt -> reggelt)
func lastWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return f[len(f)-1]
}

// pick kiest de variant voor de klinkergroep; met twee varianten (-ban/-ben) telt rond als voor
func pick(h Harmony, forms ...string) string {
	if len(forms) == 1 {
		return forms[0]
	}
	if len(forms) == 2 && h == Rounded {
		return forms[1]
	}
	return forms[h]
}

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}

func dropLast(s string) string {
	_, n := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-n]
}

// Medeklinkers die uit twee of drie letters bestaan; verdubbeld wordt alleen de eerste letter
// (busz -> busszal, hegy -> heggyel)
var digraphs = []string{"dzs", "cs", "dz", "gy", "ly", "ny", "sz", "ty", "zs"}

// finalConsonant geeft de laatste medeklinker (als klank, dus "sz" of "gy") en of die al dubbel
// is (toll, busz: nee; tollal -> "ll" ja). Leeg als het woord op een klinker eindigt.
func finalConsonant(w string) (cons string, double bool) {
	if w == "" || isVowel(lastRune(w)) {
		return "", false
	}
	for _, d := range digraphs {
		if strings.HasSuffix(w, d) {
			doubled := d[:1] + d
			return d, strings.HasSuffix(w, doubled)
		}
	}
	c := string(lastRune(w))
	return c, strings.HasSuffix(w, c+c)
}

// doubleFinal verdubbelt de laatste medeklinker: kanál -> kanáll, busz -> bussz
func doubleFinal(w, cons string) string {
	return strings.TrimSuffix(w, cons) + cons[:1] + cons
}

// consonantCluster: eindigt het woord op twee verschillende medeklinkers (mond, ért, játsz)?
// Een dubbele medeklinker (áll) telt niet.
func consonantCluster(w string) bool {
	last, double := finalConsonant(w)
	if last == "" || double {
		return false
	}
	before := strings.TrimSuffix(w, last)
	return before != "" && !isVowel(lastRune(before))
}

// syllables telt de klinkers
func syllables(w string) int {
	n := 0
	for _, r := range w {
		if isVowel(r) {
			n++
		}
	}
	return n
}
