// Uitgangen achter zelfstandige naamwoorden: naamvallen, meervoud en lijdend voorwerp.

package grammar

import (
	"strings"
)

// Form is een gemaakte woordvorm met de stappen waarmee hij gemaakt is
type Form struct {
	Text  string   `json:"text"`           // de vorm, bijv. házban
	Also  []string `json:"also,omitempty"` // ook goed (bijv. hotelban naast hotelben)
	Steps []string `json:"steps"`          // uitleg, stap voor stap
}

type suffixKind int

const (
	plain       suffixKind = iota // stam + uitgang: házban
	superessive                   // -n na een klinker, anders -on/-en/-ön
	assimilate                    // de v van -val/-vá wordt de laatste medeklinker: házzal
	adjective                     // -i: budapesti, zonder verlenging
)

// caseSuffix is een uitgang met zijn varianten (achter, voor[, rond])
type caseSuffix struct {
	forms    []string
	kind     suffixKind
	lengthen bool // eind-a/e wordt á/é: alma -> almában
}

// lengthen: een woord op -a of -e krijgt vóór een uitgang een lange klinker (alma -> almá-)
func lengthen(w string) (string, string) {
	switch lastRune(w) {
	case 'a':
		return dropLast(w) + "á", "eindigt op -a: vóór de uitgang wordt dat -á (" + w + " → " + dropLast(w) + "á-)"
	case 'e':
		return dropLast(w) + "é", "eindigt op -e: vóór de uitgang wordt dat -é (" + w + " → " + dropLast(w) + "é-)"
	}
	return w, ""
}

func harmonyStep(word string) (Harmony, bool, string) {
	h, mixed, why := HarmonyOf(word)
	return h, mixed, "klinkers van " + word + ": " + why
}

// joinForms: "-ban/-ben"
func joinForms(forms []string) string {
	return "-" + strings.Join(forms, "/-")
}

// applySuffix maakt word + uitgang
func applySuffix(word string, s caseSuffix) (Form, bool) {
	if word == "" || strings.ContainsAny(word, " -") {
		return Form{}, false // woordgroepen en woorden met een streepje laten we liggen
	}
	h, mixed, why := harmonyStep(word)
	steps := []string{why}
	build := func(h Harmony) string {
		stem := word
		if s.lengthen {
			stem, _ = lengthen(word)
		}
		switch s.kind {
		case superessive:
			if isVowel(lastRune(word)) {
				return stem + "n"
			}
			return word + pick(h, "on", "en", "ön")
		case assimilate:
			if isVowel(lastRune(word)) {
				return stem + pick(h, s.forms...)
			}
			cons, double := finalConsonant(word)
			base := word
			if !double {
				base = doubleFinal(word, cons)
			}
			return base + strings.TrimPrefix(pick(h, s.forms...), "v")
		case adjective:
			w := strings.ToLower(word)
			if lastRune(w) == 'i' {
				return w
			}
			return w + "i"
		}
		return stem + pick(h, s.forms...)
	}

	text := build(h)
	if len(s.forms) > 1 && s.kind != adjective {
		steps = append(steps, "dus "+suffixFor(h, s, word)+" (van "+joinForms(s.forms)+")")
	}
	if s.lengthen && s.kind != adjective {
		if _, step := lengthen(word); step != "" && !(s.kind == superessive && !isVowel(lastRune(word))) {
			steps = append(steps, step)
		}
	}
	switch s.kind {
	case superessive:
		if isVowel(lastRune(word)) {
			steps = append(steps, "eindigt op een klinker: alleen -n")
		} else {
			steps = append(steps, "eindigt op een medeklinker: met verbindingsklinker ("+pick(h, "-on", "-en", "-ön")+")")
		}
	case assimilate:
		if !isVowel(lastRune(word)) {
			cons, double := finalConsonant(word)
			if double {
				steps = append(steps, "de v wordt gelijk aan de laatste medeklinker; "+cons+" is al dubbel, dus geen derde")
			} else {
				steps = append(steps, "na een medeklinker valt de v weg en wordt de laatste medeklinker dubbel ("+cons+" → "+cons[:1]+cons+")")
			}
		}
	case adjective:
		if lastRune(strings.ToLower(word)) == 'i' {
			steps = append(steps, "eindigt al op -i: geen tweede i")
		} else {
			steps = append(steps, "-i erachter, zonder verlenging en met kleine letter")
		}
	}
	f := Form{Text: text, Steps: steps}
	if mixed && len(s.forms) > 1 {
		other := Back
		if h == Back {
			other = Front
		}
		if alt := build(other); alt != text {
			f.Also = []string{alt}
			f.Steps = append(f.Steps, "ook goed: "+alt)
		}
	}
	return f, true
}

func suffixFor(h Harmony, s caseSuffix, word string) string {
	if s.kind == superessive {
		return "-" + pick(h, "on", "en", "ön")
	}
	return "-" + pick(h, s.forms...)
}

// ---------- Lijdend voorwerp (-t) en meervoud (-k) ----------

// nounException: woorden die hun stam veranderen of een andere verbindingsklinker nemen
// superessive: leeg = gewoon volgens de regel, "?" = niet zeker (niet oefenen), anders de vorm
type nounException struct {
	acc, plural, superessive string
	note                     string
}

// De uitzonderingen. Lage-klinkerstammen (ház -> házat, házak), stammen die korter worden
// (madár -> madarat), v-stammen (ló -> lovat) en stammen die een klinker verliezen (sarok -> sarkot).
var nounExceptions = map[string]nounException{
	"haj":     {"hajat", "hajak", "", "neemt -a- als verbindingsklinker"},
	"lyuk":    {"lyukat", "lyukak", "", "neemt -a- als verbindingsklinker"},
	"vonal":   {"vonalat", "vonalak", "", "neemt -a- als verbindingsklinker"},
	"szakáll": {"szakállat", "szakállak", "", "neemt -a- als verbindingsklinker"},
	"társ":    {"társat", "társak", "", "neemt -a- als verbindingsklinker"},
	"tár":     {"tárat", "tárak", "", "neemt -a- als verbindingsklinker"},
	"hely":    {"helyet", "helyek", "", "neemt -e- (geen -t direct na ly)"},
	"hét":     {"hetet", "hetek", "", "de é wordt kort"},
	"tér":     {"teret", "terek", "", "de é wordt kort"},
	"úr":      {"urat", "urak", "", "de ú wordt kort en er komt -a- bij"},
	"kút":     {"kutat", "kutak", "", "de ú wordt kort en er komt -a- bij"},
	"titok":   {"titkot", "titkok", "titkon", "de o valt weg: titok → titk-"},
	"fakanál": {"fakanalat", "fakanalak", "", "samenstelling met kanál: de á wordt kort en er komt -a- bij"},
	"étterem": {"éttermet", "éttermek", "?", "de e valt weg: étterem → étterm-"},
	"könyv":   {"könyvet", "könyvek", "", "neemt -e- als verbindingsklinker, niet -ö-"},
	"föld":    {"földet", "földek", "", "neemt -e- als verbindingsklinker, niet -ö-"},
	"fül":     {"fület", "fülek", "", "neemt -e- als verbindingsklinker"},
	"ház":     {"házat", "házak", "", "neemt -a- als verbindingsklinker"},
	"fal":     {"falat", "falak", "", "neemt -a- als verbindingsklinker"},
	"hal":     {"halat", "halak", "", "neemt -a- als verbindingsklinker"},
	"toll":    {"tollat", "tollak", "", "neemt -a- als verbindingsklinker"},
	"vaj":     {"vajat", "vajak", "", "neemt -a- als verbindingsklinker"},
	"fej":     {"fejet", "fejek", "", "neemt -e- als verbindingsklinker"},
	"száj":    {"szájat", "szájak", "", "neemt -a- als verbindingsklinker"},
	"ágy":     {"ágyat", "ágyak", "", "neemt -a- als verbindingsklinker"},
	"fog":     {"fogat", "fogak", "", "neemt -a- als verbindingsklinker"},
	"hold":    {"holdat", "holdak", "", "neemt -a- als verbindingsklinker"},
	"vár":     {"várat", "várak", "", "neemt -a- als verbindingsklinker"},
	"gyár":    {"gyárat", "gyárak", "", "neemt -a- als verbindingsklinker"},
	"nyár":    {"nyarat", "nyarak", "", "de á wordt kort en er komt -a- bij"},
	"út":      {"utat", "utak", "", "de ú wordt kort en er komt -a- bij"},
	"híd":     {"hidat", "hidak", "", "de í wordt kort en er komt -a- bij"},
	"madár":   {"madarat", "madarak", "", "de á wordt kort en er komt -a- bij"},
	"kanál":   {"kanalat", "kanalak", "", "de á wordt kort en er komt -a- bij"},
	"pohár":   {"poharat", "poharak", "", "de á wordt kort en er komt -a- bij"},
	"szamár":  {"szamarat", "szamarak", "", "de á wordt kort en er komt -a- bij"},
	"kenyér":  {"kenyeret", "kenyerek", "", "de é wordt kort"},
	"levél":   {"levelet", "levelek", "", "de é wordt kort"},
	"egér":    {"egeret", "egerek", "", "de é wordt kort"},
	"tél":     {"telet", "telek", "", "de é wordt kort"},
	"név":     {"nevet", "nevek", "", "de é wordt kort"},
	"kéz":     {"kezet", "kezek", "", "de é wordt kort"},
	"víz":     {"vizet", "vizek", "", "de í wordt kort"},
	"tűz":     {"tüzet", "tüzek", "", "de ű wordt kort"},
	"ló":      {"lovat", "lovak", "lovon", "v-stam: ló → lov-"},
	"tó":      {"tavat", "tavak", "tavon", "v-stam: tó → tav-"},
	"hó":      {"havat", "havak", "havon", "v-stam: hó → hav-"},
	"kő":      {"követ", "kövek", "kövön", "v-stam: kő → köv-"},
	"fű":      {"füvet", "füvek", "füvön", "v-stam: fű → füv-"},
	"mű":      {"művet", "művek", "művön", "v-stam: mű → műv-"},
	"szó":     {"szót", "szavak", "szavon", "meervoud met v-stam: szavak"},
	"sarok":   {"sarkot", "sarkok", "sarkon", "de o valt weg: sarok → sark-"},
	"bokor":   {"bokrot", "bokrok", "bokron", "de o valt weg: bokor → bokr-"},
	"cukor":   {"cukrot", "cukrok", "cukron", "de o valt weg: cukor → cukr-"},
	"szobor":  {"szobrot", "szobrok", "szobron", "de o valt weg: szobor → szobr-"},
	"tükör":   {"tükröt", "tükrök", "tükrön", "de ö valt weg: tükör → tükr-"},
	"álom":    {"álmot", "álmok", "?", "de o valt weg: álom → álm-"},
	"majom":   {"majmot", "majmok", "majmon", "de o valt weg: majom → majm-"},
	"fogalom": {"fogalmat", "fogalmak", "?", "de o valt weg en er komt -a- bij: fogalm-"},
	"dolog":   {"dolgot", "dolgok", "dolgon", "de o valt weg: dolog → dolg-"},
	"terem":   {"termet", "termek", "?", "de e valt weg: terem → term-"},
	"eper":    {"epret", "eprek", "?", "de e valt weg: eper → epr-"},
	"torony":  {"tornyot", "tornyok", "tornyon", "de o valt weg: torony → torny-"},
	"selyem":  {"selymet", "selymek", "?", "de e valt weg: selyem → selym-"},
	"lélek":   {"lelket", "lelkek", "?", "de é wordt kort en de e valt weg: lelk-"},
	"teher":   {"terhet", "terhek", "?", "de h en r wisselen: terh-"},
	"falu":    {"falut", "falvak", "", "meervoud met v-stam: falvak"},
}

// Eenlettergrepige woorden en woorden met een korte laatste klinker die gewoon regelmatig zijn.
// Andere zulke woorden (nog niet in deze lijst of de uitzonderingen) oefenen we niet: daar zitten
// veel stammen tussen die veranderen (ház -> házat, sarok -> sarkot), en raden doen we niet.
// Samenstellingen volgen hun laatste deel (belváros als város, kórház als ház).
var regularNouns = map[string]bool{}

func registerRegular(words ...string) {
	for _, w := range words {
		regularNouns[w] = true
	}
}

func init() {
	registerRegular(
		"alak", "arc", "asztal", "bank", "város", "beteg", "bolt", "busz", "citrom", "csend", "csütörtök", "cím",
		"dokumentum", "dzsip", "dzsungel", "szünet", "eszköz", "fejezet", "feladat", "felület", "film", "fjord", "orvos",
		"fotel", "másolat", "fókusz", "főnök", "gyakorlat", "gyalogos", "gyerek", "gép", "hang", "hegy", "helyzet", "nap",
		"hónap", "hőmérséklet", "internet", "jog", "kedd", "kozmetikus", "kukac", "kép", "laptop", "lány", "jegy", "telefon",
		"mondat", "motor", "füzet", "mérséklet", "szak", "nyelv", "nyelvtan", "nyugat", "park", "perc", "program", "udvar",
		"péntek", "rag", "reggel", "sport", "szombat", "szám", "szín", "sör", "templom", "tenger", "természet", "tyúk",
		"típus", "tök", "vicc", "villamos", "vonat", "xilofon", "áram", "szer", "épület", "év", "üveg", "üzlet", "kert",
		"kés", "pénz", "szék", "ablak", "húg", "amszterdam", "főváros", "rész", "barát", "sor", "kör", "bor", "tej",
	)
}

// lookupSuffix zoekt een woord in een lijst, ook als laatste deel van een samenstelling
// (kórház -> ház). Het eerste deel moet minstens drie letters hebben (gitár is geen gi + tár), en
// woorden op een klinker tellen niet als laatste deel (autó is geen au + tó).
func lookupSuffix[V any](m map[string]V, w string) (key string, v V, ok bool) {
	if v, ok := m[w]; ok {
		return w, v, true
	}
	best := ""
	for k := range m {
		if len(k) > len(best) && strings.HasSuffix(w, k) && !isVowel(lastRune(k)) &&
			len([]rune(strings.TrimSuffix(w, k))) >= 3 {
			best = k
		}
	}
	if best == "" {
		return "", v, false
	}
	return best, m[best], true
}

// riskyStem: eenlettergrepig op een medeklinker, of een korte klinker tussen twee losse
// medeklinkers aan het eind (sarok, bokor, asztal): daar zitten de stamveranderingen
func riskyStem(w string) bool {
	if isVowel(lastRune(w)) {
		return false
	}
	if syllables(w) <= 1 {
		return true
	}
	cons, double := finalConsonant(w)
	if double {
		return false
	}
	before := strings.TrimSuffix(w, cons)
	v := lastRune(before)
	if !strings.ContainsRune("aeoöuü", v) {
		return false // lange klinker of medeklinkercluster: regelmatig (behalve de lijst: madár, kenyér)
	}
	before = dropLast(before)
	return before != "" && !isVowel(lastRune(before))
}

// directT: na deze medeklinkers komt de -t zonder verbindingsklinker (asztalt, pénzt, lányt)
var directT = []string{"j", "l", "ly", "n", "ny", "r", "s", "sz", "z", "zs"}

func accOrPlural(word string, plural bool) (Form, bool) {
	if word == "" || strings.ContainsAny(word, " -") {
		return Form{}, false
	}
	w := strings.ToLower(word)
	if key, ex, ok := lookupSuffix(nounExceptions, w); ok {
		prefix := strings.TrimSuffix(w, key)
		text := prefix + ex.acc
		if plural {
			text = prefix + ex.plural
		}
		steps := []string{word + " is een uitzondering"}
		if prefix != "" {
			steps = []string{"samenstelling: het laatste deel " + key + " is een uitzondering"}
		}
		if ex.note != "" {
			steps = append(steps, ex.note)
		}
		return Form{Text: text, Steps: steps}, true
	}
	h, _, why := harmonyStep(word)
	steps := []string{why}
	if isVowel(lastRune(w)) {
		stem, step := lengthen(word)
		if step != "" {
			steps = append(steps, step)
		}
		if plural {
			return Form{Text: stem + "k", Steps: append(steps, "eindigt op een klinker: alleen -k")}, true
		}
		return Form{Text: stem + "t", Steps: append(steps, "eindigt op een klinker: alleen -t")}, true
	}
	if _, _, regular := lookupSuffix(regularNouns, w); riskyStem(w) && !regular {
		return Form{}, false
	}
	if plural {
		return Form{Text: word + pick(h, "ok", "ek", "ök"),
			Steps: append(steps, "eindigt op een medeklinker: verbindingsklinker, dus "+pick(h, "-ok", "-ek", "-ök"))}, true
	}
	cons, _ := finalConsonant(w)
	for _, d := range directT {
		if cons == d {
			return Form{Text: word + "t", Steps: append(steps, "eindigt op "+cons+": daarna mag de -t direct (na j, l, ly, n, ny, r, s, sz, z, zs)")}, true
		}
	}
	return Form{Text: word + pick(h, "ot", "et", "öt"),
		Steps: append(steps, "eindigt op "+cons+": daarna kan de -t niet direct, dus "+pick(h, "-ot", "-et", "-öt"))}, true
}

// superessiveFor: v-stammen en stammen die een klinker verliezen (lovon, sarkon) staan in de lijst
func superessiveFor(word string, s caseSuffix) (Form, bool) {
	w := strings.ToLower(word)
	if key, ex, ok := lookupSuffix(nounExceptions, w); ok && ex.superessive != "" {
		if ex.superessive == "?" {
			return Form{}, false // niet zeker: niet oefenen
		}
		return Form{Text: strings.TrimSuffix(w, key) + ex.superessive, Steps: []string{word + " is een uitzondering", ex.note}}, true
	}
	return applySuffix(word, s)
}
