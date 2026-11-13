package storage

import "slices"

// Omgekeerde richtingen. Bij koffie -> kávé hoort kávé -> koffie: het bronwoord van de een is een
// antwoord van de ander, en andersom. Die strikte regel (Of) bepaalt wat bij hetzelfde woord
// hoort: groepen, controle, verwijderen. De planning gebruikt een ruimere regel (Related): daar is
// één kant genoeg, want ook een kaart die een andere maar half raakt verklapt het antwoord.

// Reverses is een opzoektabel op bron- en antwoordwoord, om omgekeerde richtingen snel te vinden.
// Hij hoort bij één toestand van de concepten: bouw hem opnieuw na een wijziging.
type Reverses struct {
	concepts []TranslationConcept
	bySource map[string][]int
	byTarget map[string][]int
}

func NewReverses(concepts []TranslationConcept) Reverses {
	r := Reverses{concepts: concepts, bySource: map[string][]int{}, byTarget: map[string][]int{}}
	for i, c := range concepts {
		r.bySource[c.SourceWordID] = append(r.bySource[c.SourceWordID], i)
		for _, t := range c.TargetWordIDs {
			r.byTarget[t] = append(r.byTarget[t], i)
		}
	}
	return r
}

// Of geeft de posities van de omgekeerde richting(en) van c: hun bronwoord is een antwoord van c,
// en het bronwoord van c is een van hun antwoorden
func (r Reverses) Of(c TranslationConcept) []int {
	var out []int
	for _, t := range c.TargetWordIDs {
		for _, i := range r.bySource[t] {
			if o := r.concepts[i]; o.ID != c.ID && slices.Contains(o.TargetWordIDs, c.SourceWordID) && !slices.Contains(out, i) {
				out = append(out, i)
			}
		}
	}
	return out
}

// Related geeft de posities van alle andere concepten die c raken: hun bronwoord is een antwoord
// van c, of het bronwoord van c is een van hun antwoorden (zie Related voor twee concepten)
func (r Reverses) Related(c TranslationConcept) []int {
	var out []int
	add := func(i int) {
		if r.concepts[i].ID != c.ID && !slices.Contains(out, i) {
			out = append(out, i)
		}
	}
	for _, t := range c.TargetWordIDs {
		for _, i := range r.bySource[t] {
			add(i)
		}
	}
	for _, i := range r.byTarget[c.SourceWordID] {
		add(i)
	}
	return out
}

// Related is true als twee concepten elkaar raken: het bronwoord van de een is een antwoord van
// de ander (in minstens één richting). Zulke kaarten horen niet in dezelfde ronde of op dezelfde dag.
func Related(a, b TranslationConcept) bool {
	return slices.Contains(a.TargetWordIDs, b.SourceWordID) || slices.Contains(b.TargetWordIDs, a.SourceWordID)
}
