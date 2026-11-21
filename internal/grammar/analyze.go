// Vormen herkennen: welke vorm van een woord heb je getypt (voor de feedback in de quiz).

package grammar

import "hungarian-language-learner/internal/engine"

// rulesFor: de regels die op je eigen woorden van dit type werken (niet de regels met een vaste
// lijst, zoals -i en de naamval met persoon), met de personen erbij
func rulesFor(w Word) []Rule {
	var out []Rule
	for _, r := range Rules {
		if r.WordType == w.Type && (r.pool == nil || r.ID == "szor") {
			out = append(out, r)
		}
	}
	return out
}

func personsOf(r Rule) []Person {
	if !r.Persons {
		return []Person{Sg3}
	}
	return []Person{Sg1, Sg2, Sg3, Pl1, Pl2, Pl3}
}

// Match is een herkende vorm: input is word volgens Rule (bij werkwoorden in Person)
type Match struct {
	Word     string `json:"word"`
	Rule     string `json:"rule"`
	RuleName string `json:"ruleName"`
	Person   string `json:"person,omitempty"`
}

// Analyze zoekt of input een vorm van het woord is (olvasok = olvas, onbepaald, én). Eerst
// precies, anders zonder op accenten te letten. De basisvorm zelf telt niet.
func Analyze(w Word, input string) (Match, bool) {
	in := norm(input)
	if in == "" || in == norm(w.Hungarian) {
		return Match{}, false
	}
	for _, loose := range []bool{false, true} {
		for _, r := range rulesFor(w) {
			for _, p := range personsOf(r) {
				f, ok := r.MakeFor(w, p)
				if !ok {
					continue
				}
				for _, form := range append([]string{f.Text}, f.Also...) {
					if in == norm(form) || (loose && engine.Fold(in) == engine.Fold(form)) {
						m := Match{Word: w.Hungarian, Rule: r.ID, RuleName: r.Name}
						if r.Persons {
							m.Person = PersonLabels[p]
						}
						return m, true
					}
				}
			}
		}
	}
	return Match{}, false
}
