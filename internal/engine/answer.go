package engine

import "strings"

// accentReplacer haalt de Hongaarse (en Nederlandse) accenten weg voor een "bijna goed"-vergelijking
var accentReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ö", "o", "ő", "o", "ú", "u", "ü", "u", "ű", "u",
	"ë", "e", "ï", "i", "è", "e",
)

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// Fold maakt tekst vergelijkbaar voor zoeken en sorteren: kleine letters, zonder accenten
func Fold(s string) string {
	return accentReplacer.Replace(normalize(s))
}

// CheckAnswer vergelijkt de invoer met de geaccepteerde antwoorden.
// accentOnly is true als het antwoord fout is, maar alleen in de accenten afwijkt.
func CheckAnswer(input string, accepted []string) (correct bool, accentOnly bool) {
	in := normalize(input)
	if in == "" {
		return false, false
	}
	for _, a := range accepted {
		if in == normalize(a) {
			return true, false
		}
	}
	for _, a := range accepted {
		if accentReplacer.Replace(in) == accentReplacer.Replace(normalize(a)) {
			return false, true
		}
	}
	return false, false
}
