package engine

import "strings"

// accentReplacer haalt de Hongaarse (en Nederlandse) accenten weg voor een "bijna goed"-vergelijking
var accentReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ö", "o", "ő", "o", "ú", "u", "ü", "u", "ű", "u",
	"ë", "e", "ï", "i", "è", "e",
)

// dashReplacer maakt van elk soort streepje het gewone streepje van het toetsenbord
var dashReplacer = strings.NewReplacer("–", "-", "—", "-", "‐", "-", "‑", "-", "−", "-")

// Clean maakt ingevoerde tekst netjes: geen spaties eromheen, en lange streepjes (tai–csi)
// worden een gewoon streepje (tai-csi), zodat je het antwoord met je toetsenbord kunt typen
func Clean(s string) string {
	return strings.TrimSpace(dashReplacer.Replace(s))
}

// normalize maakt een antwoord vergelijkbaar: kleine letters, enkele spaties, gewone streepjes
// en zonder leestekens aan het eind (mi = mi?)
func normalize(s string) string {
	s = strings.Join(strings.Fields(strings.ToLower(dashReplacer.Replace(s))), " ")
	return strings.TrimRight(s, "?!.… ")
}

// Fold maakt tekst vergelijkbaar voor zoeken en sorteren: kleine letters, zonder accenten
func Fold(s string) string {
	return accentReplacer.Replace(normalize(s))
}

// CheckAnswer vergelijkt de invoer met de geaccepteerde antwoorden.
// accentOnly is true als het antwoord fout is, maar alleen in de accenten afwijkt.
// Een deel tussen haakjes is optioneel, met / voor alternatieven: "(hij/zij) leest"
// accepteert "leest", "hij leest", "zij leest" en "hij/zij leest".
func CheckAnswer(input string, accepted []string) (correct bool, accentOnly bool) {
	in := normalize(input)
	if in == "" {
		return false, false
	}
	var variants []string
	for _, a := range accepted {
		variants = append(variants, Variants(a)...)
	}
	for _, v := range variants {
		if in == normalize(v) {
			return true, false
		}
	}
	for _, v := range variants {
		if accentReplacer.Replace(in) == accentReplacer.Replace(normalize(v)) {
			return false, true
		}
	}
	return false, false
}

// Variants geeft alle schrijfwijzen van een antwoord met optionele delen tussen haakjes:
// het antwoord zelf, zonder het deel, en met elk alternatief uit het deel.
func Variants(answer string) []string {
	open := strings.Index(answer, "(")
	end := strings.Index(answer, ")")
	if open < 0 || end < open {
		return []string{answer}
	}
	inner, rest := answer[open+1:end], answer[end+1:]
	prefix := answer[:open]

	options := append([]string{"", inner}, strings.Split(inner, "/")...)
	out := []string{strings.Join(strings.Fields(answer), " ")} // ook letterlijk, met haakjes
	for _, opt := range options {
		for _, tail := range Variants(rest) { // ook latere haakjes
			v := strings.Join(strings.Fields(prefix+" "+opt+" "+tail), " ")
			if v != "" && !contains(out, v) {
				out = append(out, v)
			}
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
