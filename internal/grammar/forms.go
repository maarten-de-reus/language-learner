// Alle vormen van een woord op een rij, voor Controleren: zo ziet de controleur wat de regels van
// een nieuw woord maken, en kan hij een verkeerde vorm melden voordat die geoefend wordt.

package grammar

import "strings"

// FormRow is één vorm van een woord in het overzicht
type FormRow struct {
	Rule     string `json:"rule"`
	RuleName string `json:"ruleName"`
	Person   int    `json:"person"`
	Label    string `json:"label"`          // bijv. "-ban/-ben" of "én (ik)"
	Key      string `json:"key"`            // sleutel voor afwijkende vormen en twijfels
	Text     string `json:"text,omitempty"` // leeg als de regels het niet zeker weten
	Fixed    bool   `json:"fixed"`          // afwijkende vorm bij het woord
	Doubt    string `json:"doubt,omitempty"`
}

// AllForms: alle vormen van een woord volgens de regels, ook de vormen die de regels overslaan
// (Text leeg) en de vormen waar een twijfel op staat
func AllForms(w Word) []FormRow {
	var rows []FormRow
	for _, r := range rulesFor(w) {
		for _, p := range personsOf(r) {
			key := FormKey(r.ID, p)
			row := FormRow{Rule: r.ID, RuleName: r.Name, Person: int(p), Label: r.Name, Key: key, Doubt: w.Doubts[key]}
			if r.Persons {
				row.Label = PersonLabels[p]
			}
			if v, ok := w.Forms[key]; ok {
				row.Fixed = true
				if v != "?" {
					row.Text = v
				}
			} else if f, ok := r.Make(w.Hungarian, p); ok {
				row.Text = f.Text
			}
			if r.ID == "ww-bep" && w.Forms["bep"] == "?" {
				row.Text, row.Fixed = "", true
			}
			rows = append(rows, row)
		}
	}
	return rows
}

// IsFormKey: bestaat deze sleutel voor afwijkende vormen en twijfels?
func IsFormKey(key string) bool {
	_, ok := formKeys[strings.ToLower(key)]
	return ok
}
