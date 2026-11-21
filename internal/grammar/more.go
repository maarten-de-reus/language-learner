// Meer regels: -kor (tijdstip), -szor/-szer/-ször (keer), vergelijken (-bb, leg-…-bb) en de
// naamvallen met een persoon (nálam, velem).

package grammar

import "strings"

// ---------- -kor ----------

// korWords: -kor gaat achter tijdstippen: klokuren en feestdagen
var korWords = []iWord{
	{hu: "egy", nl: "één (uur)"}, {hu: "kettő", nl: "twee (uur)"}, {hu: "három", nl: "drie (uur)"},
	{hu: "négy", nl: "vier (uur)"}, {hu: "öt", nl: "vijf (uur)"}, {hu: "hat", nl: "zes (uur)"},
	{hu: "hét", nl: "zeven (uur)"}, {hu: "nyolc", nl: "acht (uur)"}, {hu: "kilenc", nl: "negen (uur)"},
	{hu: "tíz", nl: "tien (uur)"}, {hu: "tizenegy", nl: "elf (uur)"}, {hu: "tizenkettő", nl: "twaalf (uur)"},
	{hu: "éjfél", nl: "middernacht"}, {hu: "karácsony", nl: "Kerstmis"}, {hu: "húsvét", nl: "Pasen"},
	{hu: "szilveszter", nl: "oudejaarsavond"}, {hu: "vacsora", nl: "avondeten"}, {hu: "ebéd", nl: "middageten"},
	{hu: "reggeli", nl: "ontbijt"}, {hu: "napkelte", nl: "zonsopgang"}, {hu: "napnyugta", nl: "zonsondergang"},
}

func korForm(word string) (Form, bool) {
	if word == "" || strings.ContainsAny(word, " -") {
		return Form{}, false
	}
	return Form{Text: word + "kor", Steps: []string{
		"-kor is voor alle woorden hetzelfde: geen klinkerharmonie, en een eind-a/-e blijft kort (vacsorakor)",
	}}, true
}

// ---------- -szor/-szer/-ször ----------

var szorWords = []iWord{
	{hu: "egy", nl: "één"}, {hu: "kettő", nl: "twee", form: "kétszer"}, {hu: "három", nl: "drie"},
	{hu: "négy", nl: "vier"}, {hu: "öt", nl: "vijf"}, {hu: "hat", nl: "zes"}, {hu: "hét", nl: "zeven"},
	{hu: "nyolc", nl: "acht"}, {hu: "kilenc", nl: "negen"}, {hu: "tíz", nl: "tien"}, {hu: "húsz", nl: "twintig"},
	{hu: "száz", nl: "honderd"}, {hu: "ezer", nl: "duizend"}, {hu: "hány", nl: "hoeveel"},
	{hu: "sok", nl: "veel"}, {hu: "néhány", nl: "enkele"}, {hu: "több", nl: "meer"},
}

// szorPool: de vaste lijst plus je eigen telwoorden (huszonöt, negyvenegy …)
func szorPool(words []Word) []Word {
	out := poolOf(szorWords, "szor", words)
	seen := map[string]bool{}
	for _, w := range out {
		seen[strings.ToLower(w.Hungarian)] = true
	}
	for _, w := range words {
		h := strings.ToLower(w.Hungarian)
		if w.Type != "telw" || seen[h] || h == "két" || strings.Contains(h, "fél") || strings.Contains(h, "negyed") {
			continue
		}
		seen[h] = true
		out = append(out, w)
	}
	return out
}

// ---------- vergelijken ----------

// comparativeExceptions: trappen van vergelijking die de regel niet zelf kan maken
var comparativeExceptions = map[string]string{
	"jó": "jobb", "sok": "több", "kicsi": "kisebb", "kis": "kisebb", "szép": "szebb", "nagy": "nagyobb",
	"öreg": "idősebb", "kevés": "kevesebb", "nehéz": "nehezebb", "bő": "bővebb", "régi": "régebbi",
	"hosszú": "hosszabb", "lassú": "lassabb", "könnyű": "könnyebb", "ifjú": "ifjabb", "bátor": "bátrabb",
	"keserű": "keserűbb", "szomorú": "szomorúbb", "savanyú": "savanyúbb", "gyönyörű": "gyönyörűbb",
	"egyszerű": "egyszerűbb", "hű": "hűbb",
}

// Bijvoeglijke naamwoorden zonder trappen van vergelijking (links, Hongaars, …)
var notComparable = map[string]bool{
	"bal": true, "jobb": true, "magyar": true, "holland": true, "spanyol": true, "angol": true, "német": true,
	"francia": true, "olasz": true, "egyéb": true, "külön": true, "fél": true, "utolsó": true, "első": true,
	"kedvenc": true, "normális": true,
}

// Comparative maakt de vergrotende trap: -bb, met verbindingsklinker -abb/-ebb na een medeklinker
func Comparative(adj string) (Form, bool) {
	w := strings.ToLower(strings.TrimSpace(adj))
	if w == "" || strings.ContainsAny(w, " -") || notComparable[w] || strings.HasSuffix(w, "bb") || strings.HasSuffix(w, "bbi") {
		return Form{}, false
	}
	if ex, ok := comparativeExceptions[w]; ok {
		return Form{Text: ex, Steps: []string{w + " is een uitzondering"}}, true
	}
	switch last := lastRune(w); {
	case last == 'a' || last == 'e':
		stem, step := lengthen(w)
		return Form{Text: stem + "bb", Steps: []string{step, "na een klinker alleen -bb"}}, true
	case strings.ContainsRune("iúű", last):
		return Form{}, false // régi -> régebbi, hosszú -> hosszabb maar keserű -> keserűbb: niet zeker
	case isVowel(last):
		return Form{Text: w + "bb", Steps: []string{"eindigt op een klinker: alleen -bb"}}, true
	}
	h, _, why := HarmonyOf(w)
	e := pick(h, "abb", "ebb")
	return Form{Text: w + e, Steps: []string{"klinkers van " + w + ": " + why, "na een medeklinker met verbindingsklinker: -" + e}}, true
}

// Superlative: leg- + vergrotende trap (legnagyobb)
func Superlative(adj string) (Form, bool) {
	f, ok := Comparative(adj)
	if !ok {
		return Form{}, false
	}
	return Form{Text: "leg" + f.Text, Steps: append(f.Steps, "overtreffende trap: leg- ervoor ("+f.Text+" → leg"+f.Text+")")}, true
}

// ---------- naamvallen met een persoon ----------

// personalCases: de naamval met persoonsuitgang (nálam = bij mij). Dit is een vaste tabel.
var personalCases = []struct {
	key, nl string
	forms   [6]string
}{
	{"-nak/-nek", "aan, voor", [6]string{"nekem", "neked", "neki", "nekünk", "nektek", "nekik"}},
	{"-nál/-nél", "bij", [6]string{"nálam", "nálad", "nála", "nálunk", "nálatok", "náluk"}},
	{"-tól/-től", "van (vandaan)", [6]string{"tőlem", "tőled", "tőle", "tőlünk", "tőletek", "tőlük"}},
	{"-val/-vel", "met", [6]string{"velem", "veled", "vele", "velünk", "veletek", "velük"}},
	{"-hoz/-hez/-höz", "naar (toe)", [6]string{"hozzám", "hozzád", "hozzá", "hozzánk", "hozzátok", "hozzájuk"}},
	{"-ban/-ben", "in", [6]string{"bennem", "benned", "benne", "bennünk", "bennetek", "bennük"}},
	{"-ból/-ből", "uit", [6]string{"belőlem", "belőled", "belőle", "belőlünk", "belőletek", "belőlük"}},
	{"-ba/-be", "in (naar binnen)", [6]string{"belém", "beléd", "belé", "belénk", "belétek", "beléjük"}},
	{"-n/-on/-en/-ön", "op", [6]string{"rajtam", "rajtad", "rajta", "rajtunk", "rajtatok", "rajtuk"}},
	{"-ról/-ről", "over, van … af", [6]string{"rólam", "rólad", "róla", "rólunk", "rólatok", "róluk"}},
	{"-ra/-re", "op (naar)", [6]string{"rám", "rád", "rá", "ránk", "rátok", "rájuk"}},
	{"-ért", "voor, om", [6]string{"értem", "érted", "érte", "értünk", "értetek", "értük"}},
}

func personalPool(words []Word) []Word {
	out := make([]Word, len(personalCases))
	for i, pc := range personalCases {
		out[i] = Word{Hungarian: pc.key, Dutch: pc.nl, Type: "vnw"}
	}
	return out
}

func personalForm(key string, p Person) (Form, bool) {
	for _, pc := range personalCases {
		if pc.key == key {
			return Form{Text: pc.forms[p], Steps: []string{
				"de naamval krijgt een persoonsuitgang: -m, -d, –, -unk/-ünk, -tok/-tek, -uk/-ük",
				"alle zes: " + strings.Join(pc.forms[:], ", "),
			}}, true
		}
	}
	return Form{}, false
}
