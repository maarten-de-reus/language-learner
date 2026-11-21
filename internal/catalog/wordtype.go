// Woordsoorten: welke regels bij een woord horen (vervoegen bij werkwoorden, naamvallen bij
// zelfstandige naamwoorden). De woordsoort hoort bij het woord, dus bij beide richtingen.

package catalog

import (
	"fmt"
	"maps"
	"strings"

	"hungarian-language-learner/internal/storage"
)

// WordType is een woordsoort met zijn korte code (zoals in woordenboeken) en naam
type WordType struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// WordTypes zijn alle woordsoorten, in de volgorde van de keuzelijst
var WordTypes = []WordType{
	{"zn", "zelfstandig naamwoord"},
	{"ww", "werkwoord"},
	{"bn", "bijvoeglijk naamwoord"},
	{"bw", "bijwoord"},
	{"vnw", "voornaamwoord"},
	{"telw", "telwoord"},
	{"lw", "lidwoord"},
	{"vw", "voegwoord"},
	{"vz", "achterzetsel"},
	{"tw", "tussenwerpsel"},
	{"uitdr", "uitdrukking"},
}

// ParseWordType herkent een code of naam (hoofdletters maken niet uit, "werkwoord" = "ww").
// Leeg blijft leeg: woordsoort onbekend.
func ParseWordType(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", nil
	}
	for _, t := range WordTypes {
		if s == t.Code || s == t.Name {
			return t.Code, nil
		}
	}
	codes := make([]string, len(WordTypes))
	for i, t := range WordTypes {
		codes[i] = t.Code
	}
	return "", fmt.Errorf("onbekende woordsoort %q: gebruik %s (of de volledige naam, bijv. werkwoord)", s, strings.Join(codes, ", "))
}

// setWordType geeft concept i en zijn omgekeerde richting(en) deze woordsoort. Met onlyEmpty
// worden alleen woorden zonder woordsoort ingevuld (bij een import van bestaande woorden).
// true als er iets veranderd is.
func setWordType(lib *storage.Library, i int, code string, onlyEmpty bool) bool {
	if code == "" {
		return false
	}
	changed := false
	for _, j := range append([]int{i}, storage.NewReverses(lib.Concepts).Of(lib.Concepts[i])...) {
		c := &lib.Concepts[j]
		if c.WordType == code || (onlyEmpty && c.WordType != "") {
			continue
		}
		c.WordType = code
		changed = true
	}
	return changed
}

// setForms geeft concept i en zijn omgekeerde richting(en) deze afwijkende vormen (zie
// grammar.ParseForms). Met onlyEmpty alleen bij woorden die er nog geen hebben.
func setForms(lib *storage.Library, i int, forms map[string]string, onlyEmpty bool) bool {
	if len(forms) == 0 {
		return false
	}
	changed := false
	for _, j := range append([]int{i}, storage.NewReverses(lib.Concepts).Of(lib.Concepts[i])...) {
		c := &lib.Concepts[j]
		if maps.Equal(c.Forms, forms) || (onlyEmpty && len(c.Forms) > 0) {
			continue
		}
		c.Forms = maps.Clone(forms)
		changed = true
	}
	return changed
}

// SetWordType zet de woordsoort van een concept en zijn omgekeerde richting(en); leeg wist hem
func SetWordType(lib *storage.Library, id, wordType string) error {
	i := lib.Index(id)
	if i < 0 {
		return ErrNotFound
	}
	code, err := ParseWordType(wordType)
	if err != nil {
		return err
	}
	if code == "" {
		for _, j := range append([]int{i}, storage.NewReverses(lib.Concepts).Of(lib.Concepts[i])...) {
			lib.Concepts[j].WordType = ""
		}
		return nil
	}
	setWordType(lib, i, code, false)
	return nil
}
