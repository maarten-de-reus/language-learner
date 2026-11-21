// Afwijkende vormen per woord: uitzonderingen die bij het woord zelf staan (zesde importkolom of
// Beheer) in plaats van in de lijsten in de code. Ze gaan vóór de regels.
//
// Schrijfwijze: sleutel=vorm, gescheiden door komma's: "t=házat, k=házak". Een ? als vorm
// betekent "niet zeker, niet oefenen": "k=?". bep=? zegt dat een werkwoord geen bepaalde
// vervoeging heeft (geen lijdend voorwerp, zoals sétál).

package grammar

import (
	"fmt"
	"sort"
	"strings"
)

type formSlot struct {
	rule   string
	person Person
}

// formKeys: sleutel -> regel (en persoon). De sleutels zijn de uitgangen zoals je ze schrijft.
var formKeys = map[string]formSlot{
	"ban": {"ban", Sg3}, "ba": {"ba", Sg3}, "ból": {"bol", Sg3}, "on": {"on", Sg3}, "ra": {"ra", Sg3},
	"ról": {"rol", Sg3}, "nál": {"nal", Sg3}, "hoz": {"hoz", Sg3}, "tól": {"tol", Sg3}, "nak": {"nak", Sg3},
	"val": {"val", Sg3}, "vá": {"va", Sg3}, "ig": {"ig", Sg3}, "ért": {"ert", Sg3}, "ként": {"kent", Sg3},
	"t": {"t", Sg3}, "k": {"k", Sg3}, "i": {"i", Sg3}, "inf": {"ww-inf", Sg3},
	"kor": {"kor", Sg3}, "szor": {"szor", Sg3}, "bb": {"bb", Sg3}, "leg": {"leg", Sg3},
	"én": {"ww-onbep", Sg1}, "te": {"ww-onbep", Sg2}, "ő": {"ww-onbep", Sg3}, "mi": {"ww-onbep", Pl1},
	"ti": {"ww-onbep", Pl2}, "ők": {"ww-onbep", Pl3},
	"bep-én": {"ww-bep", Sg1}, "bep-te": {"ww-bep", Sg2}, "bep-ő": {"ww-bep", Sg3}, "bep-mi": {"ww-bep", Pl1},
	"bep-ti": {"ww-bep", Pl2}, "bep-ők": {"ww-bep", Pl3},
}

// Zonder accenten mag ook (bol = ból)
var keyAliases = map[string]string{"bol": "ból", "rol": "ról", "nal": "nál", "tol": "tól", "va": "vá", "ert": "ért", "kent": "ként",
	"en": "én", "o": "ő", "ok": "ők", "bep-en": "bep-én", "bep-o": "bep-ő", "bep-ok": "bep-ők"}

// FormKey is de sleutel voor een regel en persoon ("" als de regel geen afwijkende vorm kent)
func FormKey(ruleID string, p Person) string {
	for k, s := range formKeys {
		if s.rule == ruleID && (s.person == p || !isPersonRule(ruleID)) {
			return k
		}
	}
	return ""
}

func isPersonRule(id string) bool { return id == "ww-onbep" || id == "ww-bep" || id == "pers" }

// ParseForms leest "t=házat, k=házak". Leeg geeft nil.
func ParseForms(s string) (map[string]string, error) {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		if a, ok := keyAliases[key]; ok {
			key = a
		}
		if !ok || val == "" {
			return nil, fmt.Errorf("afwijkende vorm %q: schrijf sleutel=vorm, bijv. t=házat", part)
		}
		if key == "bep" {
			if val != "?" {
				return nil, fmt.Errorf("afwijkende vorm %q: bep kan alleen bep=? zijn (geen bepaalde vervoeging)", part)
			}
			out[key] = val
			continue
		}
		if _, known := formKeys[key]; !known {
			return nil, fmt.Errorf("afwijkende vorm %q: onbekende sleutel %q (bijv. t, k, ban, on, val, inf, én, te, ti, ők, bep-ő)", part, key)
		}
		if val != "?" && strings.ContainsAny(val, " ;=") {
			return nil, fmt.Errorf("afwijkende vorm %q: één woord per vorm", part)
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// FormatForms schrijft de vormen terug als tekst, in een vaste volgorde
func FormatForms(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + m[k]
	}
	return strings.Join(parts, ", ")
}

// MakeFor maakt de vorm voor een woord uit je lijst: een afwijkende vorm bij het woord gaat voor
func (r Rule) MakeFor(w Word, p Person) (Form, bool) {
	if r.ID == "ww-bep" && w.Forms["bep"] == "?" {
		return Form{}, false
	}
	if v, ok := w.Forms[FormKey(r.ID, p)]; ok {
		if v == "?" {
			return Form{}, false
		}
		return Form{Text: v, Steps: []string{w.Hungarian + " is een uitzondering: zo staat de vorm bij het woord"}}, true
	}
	return r.Make(w.Hungarian, p)
}
