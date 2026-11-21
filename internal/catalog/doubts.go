// Twijfels over vormen: iemand vindt dat een vorm die de regels maken niet klopt ("Klopt niet" in
// Regels of Controleren). Zolang de twijfel er staat wordt die vorm niet geoefend; in Controleren
// wordt hij opgelost: de vorm klopt toch, de goede vorm komt bij het woord, of niet oefenen.

package catalog

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/grammar"
	"hungarian-language-learner/internal/storage"
)

var ErrUnknownFormKey = errors.New("onbekende vorm")

// doubtTargets: het concept en zijn omgekeerde richting(en): een vorm hoort bij het woord
func doubtTargets(lib *storage.Library, i int) []int {
	return append([]int{i}, storage.NewReverses(lib.Concepts).Of(lib.Concepts[i])...)
}

// SetDoubt zet een twijfel op een vorm van het woord van concept id. form is de vorm zoals de
// regels hem maakten (voor de controleur).
func SetDoubt(lib *storage.Library, id, key, form string) error {
	i := lib.Index(id)
	if i < 0 {
		return ErrNotFound
	}
	if !grammar.IsFormKey(key) {
		return fmt.Errorf("%w %q", ErrUnknownFormKey, key)
	}
	for _, j := range doubtTargets(lib, i) {
		c := &lib.Concepts[j]
		if c.Doubts == nil {
			c.Doubts = map[string]string{}
		}
		c.Doubts[key] = strings.TrimSpace(form)
	}
	return nil
}

// Doubt-acties bij het oplossen
const (
	DoubtCorrect = "correct" // de vorm klopt toch
	DoubtFix     = "fix"     // de goede vorm komt bij het woord (afwijkende vorm)
	DoubtSkip    = "skip"    // niet oefenen
)

// ResolveDoubt lost een twijfel op
func ResolveDoubt(lib *storage.Library, id, key, action, form string) error {
	i := lib.Index(id)
	if i < 0 {
		return ErrNotFound
	}
	if _, ok := lib.Concepts[i].Doubts[key]; !ok {
		return fmt.Errorf("er staat geen twijfel op %q", key)
	}
	form = engine.Clean(form)
	switch action {
	case DoubtCorrect, DoubtSkip:
	case DoubtFix:
		if _, err := grammar.ParseForms(key + "=" + form); err != nil || form == "" {
			return fmt.Errorf("geef de goede vorm, één woord")
		}
	default:
		return fmt.Errorf("onbekende actie %q", action)
	}
	for _, j := range doubtTargets(lib, i) {
		c := &lib.Concepts[j]
		delete(c.Doubts, key)
		if len(c.Doubts) == 0 {
			c.Doubts = nil
		}
		if action == DoubtCorrect {
			continue
		}
		forms := maps.Clone(c.Forms)
		if forms == nil {
			forms = map[string]string{}
		}
		forms[key] = form
		if action == DoubtSkip {
			forms[key] = "?"
		}
		c.Forms = forms
	}
	return nil
}
