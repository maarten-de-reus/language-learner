// Package catalog beheert de woordenschat: concepten toevoegen en bewerken, met controles op
// duplicaten en betekenissen. Duplicaten worden herkend op tekst en context, nooit op ID:
// ID's zijn alleen labels (oude data heeft genummerde ID's zoals c_1 en w_hu_11).
package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"hungarian-language-learner/internal/storage"
)

var (
	ErrDuplicate        = errors.New("dit woord bestaat al met deze context en deze vertaling")
	ErrReverseAmbiguous = errors.New("er bestaat al een HU->NL concept met dit woord en deze hint; kies een andere hint")
	ErrHintNeeded       = errors.New("dit Nederlandse woord heeft al een betekenis: geef een hint die de nieuwe betekenis onderscheidt")
)

// NewPair is de invoer voor een nieuw NL->HU / HU->NL paar
type NewPair struct {
	Dutch     string
	Context   string   // hint bij NL->HU, bijv. "Meubel"
	Hungarian []string // geaccepteerde Hongaarse vertalingen; de eerste nieuwe is de bron van HU->NL
	Example   string   // optionele Hongaarse voorbeeldzin, voor beide richtingen
	Groups    []string // groepen voor beide richtingen; een nieuwe naam maakt de groep aan

	// NL->HU, als het Nederlandse woord al bestaat: óf de Hongaarse woorden toevoegen als extra goede
	// antwoorden bij een bestaande betekenis (echte synoniemen, bijv. bank (meubel) -> pad, kanapé),
	// óf een nieuwe betekenis met een eigen, onderscheidende hint (bijv. bank (geld) -> bank)
	ExtendForward string            // ID van bestaand NL->HU concept om aan te vullen; leeg = nieuw concept
	ForwardHints  map[string]string // hints voor bestaande NL->HU concepten die er nog geen hadden

	// HU->NL: óf samenvoegen met een bestaand concept met dezelfde betekenis (echte synoniemen,
	// bijv. szia -> hallo/hoi),
	// óf een nieuw concept met een eigen hint die de betekenis aanwijst (bijv. idő -> weer / tijd, of
	// ő -> hij / zij: Nederlands maakt onderscheid dat Hongaars niet maakt, dus allebei apart leren)
	MergeReverseInto string            // ID van bestaand HU->NL concept; leeg = nieuw concept
	ReverseContext   string            // hint voor het nieuwe HU->NL concept
	ExistingHints    map[string]string // hints voor bestaande HU->NL concepten die er nog geen hadden
}

// AddResult beschrijft wat AddPair heeft gedaan, voor de melding aan de gebruiker
type AddResult struct {
	Extended      bool   `json:"extended"`      // true als de Hongaarse woorden aan een bestaand NL->HU concept zijn toegevoegd
	ReverseSource string `json:"reverseSource"` // Hongaars woord dat de bron is van het HU->NL concept
	ReverseMerged bool   `json:"reverseMerged"` // true als het NL-woord aan een bestaand HU->NL concept is toegevoegd
	// De andere Hongaarse woorden (synoniemen, zoals kanapé bij pad,kanapé -> bank) krijgen ook een
	// eigen HU->NL concept. Een woord dat al een andere betekenis heeft wordt overgeslagen: of het
	// dezelfde betekenis is, kan alleen jij kiezen.
	ExtraReverse   []string `json:"extraReverse,omitempty"`
	SkippedReverse []string `json:"skippedReverse,omitempty"`
}

// ParseInput schoont de invoer op; meerdere Hongaarse vertalingen worden gescheiden door komma's
func ParseInput(dutch, context, hungarianInput string) (NewPair, error) {
	nc := NewPair{Dutch: strings.TrimSpace(dutch), Context: strings.TrimSpace(context)}
	if nc.Dutch == "" {
		return nc, errors.New("woord mag niet leeg zijn")
	}
	for _, hu := range strings.Split(hungarianInput, ",") {
		if hu = strings.TrimSpace(hu); hu != "" {
			nc.Hungarian = append(nc.Hungarian, hu)
		}
	}
	if len(nc.Hungarian) == 0 {
		return nc, errors.New("vertaling mag niet leeg zijn")
	}
	nc.ReverseContext = nc.Context
	return nc, nil
}

// ForwardCandidates geeft de bestaande NL->HU concepten met dit Nederlandse woord als bron
func ForwardCandidates(lib *storage.Library, dutch string) []int {
	var idx []int
	for i, c := range lib.Concepts {
		if matches(lib.Words, c, "nl", dutch, "hu") {
			idx = append(idx, i)
		}
	}
	return idx
}

// Existing geeft de NL->HU concepten die al alles bevatten: elk Hongaars woord is al een goed
// antwoord bij een betekenis van hetzelfde Nederlandse woord. nil als er iets nieuws in zit.
// De hint telt alleen mee bij de keuze tussen betekenissen: oud -> régi is hetzelfde woord als
// oud (dingen) -> régi, en oud -> régi, öreg zijn samen de twee betekenissen van oud. Hetzelfde
// woordpaar met een andere hint overhoren heeft geen zin. Herkend op tekst, niet op ID.
func Existing(lib *storage.Library, nc NewPair) []int {
	cands := ForwardCandidates(lib, nc.Dutch)
	// Zelfde hint gaat voor
	slices.SortStableFunc(cands, func(a, b int) int {
		same := func(i int) bool {
			return strings.EqualFold(strings.TrimSpace(lib.Concepts[i].ContextSource), nc.Context)
		}
		switch {
		case same(a) && !same(b):
			return -1
		case same(b) && !same(a):
			return 1
		}
		return 0
	})
	var found []int
	for _, hu := range nc.Hungarian {
		i := slices.IndexFunc(cands, func(i int) bool { return len(newAnswers(lib, lib.Concepts[i], []string{hu})) == 0 })
		if i < 0 {
			return nil
		}
		if !slices.Contains(found, cands[i]) {
			found = append(found, cands[i])
		}
	}
	return found
}

// IsDuplicate is true als er niets nieuws is (zie Existing)
func IsDuplicate(lib *storage.Library, nc NewPair) bool {
	return len(Existing(lib, nc)) > 0
}

// newAnswers geeft de Hongaarse woorden die nog geen antwoord van concept c zijn
func newAnswers(lib *storage.Library, c storage.TranslationConcept, hungarian []string) []string {
	known := TargetTexts(lib, c)
	var out []string
	for _, hu := range hungarian {
		if !slices.ContainsFunc(known, func(k string) bool { return strings.EqualFold(strings.TrimSpace(k), hu) }) {
			out = append(out, hu)
		}
	}
	return out
}

// ReverseSourceFor is het Hongaarse woord waarvoor een HU->NL concept gemaakt of aangevuld wordt:
// het eerste Hongaarse woord, of bij het aanvullen van een bestaande betekenis het eerste nieuwe
func ReverseSourceFor(lib *storage.Library, nc NewPair) string {
	if i := lib.Index(nc.ExtendForward); i >= 0 {
		if fresh := newAnswers(lib, lib.Concepts[i], nc.Hungarian); len(fresh) > 0 {
			return fresh[0]
		}
	}
	return nc.Hungarian[0]
}

// ExtraReverses verdeelt de nieuwe Hongaarse antwoorden naast ReverseSourceFor: create krijgt een
// eigen HU->NL concept, skip bestaat al met een andere betekenis. Een woord dat al een HU->NL
// concept met dit Nederlandse woord als antwoord heeft, hoeft niets.
func ExtraReverses(lib *storage.Library, nc NewPair) (create, skip []string) {
	fresh := nc.Hungarian
	if i := lib.Index(nc.ExtendForward); i >= 0 {
		fresh = newAnswers(lib, lib.Concepts[i], nc.Hungarian)
	}
	primary := ReverseSourceFor(lib, nc)
	seen := map[string]bool{strings.ToLower(primary): true}
	for _, hu := range fresh {
		if seen[strings.ToLower(hu)] {
			continue
		}
		seen[strings.ToLower(hu)] = true
		cands := ReverseCandidates(lib, hu)
		switch {
		case len(cands) == 0:
			create = append(create, hu)
		case !slices.ContainsFunc(cands, func(i int) bool {
			return slices.ContainsFunc(TargetTexts(lib, lib.Concepts[i]), func(t string) bool { return strings.EqualFold(t, nc.Dutch) })
		}):
			skip = append(skip, hu)
		}
	}
	return create, skip
}

// ReverseCandidates geeft de bestaande HU->NL concepten met dit Hongaarse woord als bron
func ReverseCandidates(lib *storage.Library, hungarian string) []int {
	var idx []int
	for i, c := range lib.Concepts {
		if matches(lib.Words, c, "hu", hungarian, "nl") {
			idx = append(idx, i)
		}
	}
	return idx
}

// ReverseHintTaken is true als een ander HU->NL concept met dit woord al deze hint heeft
func ReverseHintTaken(lib *storage.Library, nc NewPair, hint string) bool {
	for _, i := range ReverseCandidates(lib, ReverseSourceFor(lib, nc)) {
		existing := lib.Concepts[i].ContextSource
		if h, ok := nc.ExistingHints[lib.Concepts[i].ID]; ok {
			existing = h
		}
		if strings.EqualFold(strings.TrimSpace(existing), strings.TrimSpace(hint)) {
			return true
		}
	}
	return false
}

// checkForward controleert de keuze voor het Nederlandse woord en geeft de index van het
// concept dat aangevuld wordt (of -1 voor een nieuwe betekenis)
func checkForward(lib *storage.Library, nc NewPair) (int, error) {
	cands := ForwardCandidates(lib, nc.Dutch)
	if nc.ExtendForward != "" {
		i := lib.Index(nc.ExtendForward)
		if i < 0 || !slices.Contains(cands, i) {
			return -1, fmt.Errorf("NL->HU concept %q hoort niet bij %q", nc.ExtendForward, nc.Dutch)
		}
		if len(newAnswers(lib, lib.Concepts[i], nc.Hungarian)) == 0 {
			return -1, ErrDuplicate
		}
		return i, nil
	}
	if len(cands) == 0 {
		return -1, nil
	}
	// Nieuwe betekenis: de hint moet hem onderscheiden van alle bestaande betekenissen
	if nc.Context == "" {
		return -1, ErrHintNeeded
	}
	for _, i := range cands {
		c := lib.Concepts[i]
		hint := c.ContextSource
		if h, ok := nc.ForwardHints[c.ID]; ok {
			hint = h
		}
		hint = strings.TrimSpace(hint)
		if hint == "" {
			return -1, fmt.Errorf("het bestaande %q heeft nog geen hint; geef het er een zodat de betekenissen uit elkaar te houden zijn", nc.Dutch)
		}
		if strings.EqualFold(hint, nc.Context) {
			return -1, ErrContextTaken
		}
	}
	return -1, nil
}

// AddPair voegt het NL->HU concept toe (of vult een bestaande betekenis aan) en maakt of verrijkt
// het HU->NL concept. Alle controles gebeuren vóór er iets aangepast wordt.
// Nieuwe concepten komen in de wachtrij met nieuwe woorden.
func AddPair(lib *storage.Library, nc NewPair) (AddResult, error) {
	words := lib.Words
	res := AddResult{ReverseMerged: nc.MergeReverseInto != ""}

	// --- CONTROLES ---
	if IsDuplicate(lib, nc) {
		return res, ErrDuplicate
	}
	groups, err := NormalizeGroups(lib, nc.Groups)
	if err != nil {
		return res, err
	}
	extendIdx, err := checkForward(lib, nc)
	if err != nil {
		return res, err
	}
	res.Extended = extendIdx >= 0
	res.ReverseSource = ReverseSourceFor(lib, nc)
	if res.Extended && nc.ReverseContext == "" {
		nc.ReverseContext = lib.Concepts[extendIdx].ContextSource // hint van de betekenis die aangevuld wordt
	}
	mergeIdx := -1
	if res.ReverseMerged {
		mergeIdx = lib.Index(nc.MergeReverseInto)
		if mergeIdx < 0 || !matches(words, lib.Concepts[mergeIdx], "hu", res.ReverseSource, "nl") {
			return res, fmt.Errorf("HU->NL concept %q hoort niet bij %q", nc.MergeReverseInto, res.ReverseSource)
		}
	} else if ReverseHintTaken(lib, nc, nc.ReverseContext) {
		return res, ErrReverseAmbiguous
	}
	res.ExtraReverse, res.SkippedReverse = ExtraReverses(lib, nc)
	now := time.Now()

	// --- VERWERK DE WOORDEN IN DE POOL (bestaande woorden worden hergebruikt) ---
	dutchID := ensureWord(words, "nl", nc.Dutch)
	var huIDs []string
	for _, hu := range nc.Hungarian {
		if id := ensureWord(words, "hu", hu); !slices.Contains(huIDs, id) {
			huIDs = append(huIDs, id)
		}
	}
	reverseSourceID := ensureWord(words, "hu", res.ReverseSource)

	// --- CONCEPT 1: Nederlands -> Hongaars ---
	if res.Extended {
		c := &lib.Concepts[extendIdx]
		for _, id := range huIDs {
			if !slices.Contains(c.TargetWordIDs, id) {
				c.TargetWordIDs = append(c.TargetWordIDs, id)
			}
		}
		if c.Example == "" && nc.Example != "" {
			c.Example, c.ExampleLang = nc.Example, exampleLang(nc.Example)
		}
		addGroups(c, groups)
	} else {
		for id, hint := range nc.ForwardHints {
			if i := lib.Index(id); i >= 0 {
				lib.Concepts[i].ContextSource = strings.TrimSpace(hint)
			}
		}
		base := "c_nl_hu_" + safeID(nc.Dutch)
		if nc.Context != "" {
			base += "_" + safeID(nc.Context)
		}
		lib.Concepts = append(lib.Concepts, storage.TranslationConcept{
			ID:            uniqueConceptID(lib, base),
			SourceWordID:  dutchID,
			ContextSource: nc.Context,
			TargetWordIDs: huIDs,
			Example:       nc.Example,
			ExampleLang:   exampleLang(nc.Example),
			Groups:        slices.Clone(groups),
			NextReview:    now,
		})
	}

	// --- CONCEPT 2: Hongaars -> Nederlands ---
	if mergeIdx >= 0 {
		merged := &lib.Concepts[mergeIdx]
		if !slices.Contains(merged.TargetWordIDs, dutchID) {
			merged.TargetWordIDs = append(merged.TargetWordIDs, dutchID)
		}
		if merged.Example == "" && nc.Example != "" {
			merged.Example, merged.ExampleLang = nc.Example, exampleLang(nc.Example)
		}
		addGroups(merged, groups)
	} else {
		for id, hint := range nc.ExistingHints {
			if i := lib.Index(id); i >= 0 {
				lib.Concepts[i].ContextSource = strings.TrimSpace(hint)
			}
		}
		base := "c_hu_nl_" + safeID(res.ReverseSource)
		if nc.ReverseContext != "" {
			base += "_" + safeID(nc.ReverseContext)
		}
		lib.Concepts = append(lib.Concepts, storage.TranslationConcept{
			ID:            uniqueConceptID(lib, base),
			SourceWordID:  reverseSourceID,
			ContextSource: strings.TrimSpace(nc.ReverseContext),
			TargetWordIDs: []string{dutchID},
			Example:       nc.Example,
			ExampleLang:   exampleLang(nc.Example),
			Groups:        slices.Clone(groups),
			NextReview:    now,
		})
	}

	// --- SYNONIEMEN: elk ander nieuw Hongaars woord ook als vraag (kanapé -> bank) ---
	// Zonder voorbeeldzin: die gaat over het eerste woord en zou bij een synoniem verwarren
	for _, hu := range res.ExtraReverse {
		base := "c_hu_nl_" + safeID(hu)
		if nc.ReverseContext != "" {
			base += "_" + safeID(nc.ReverseContext)
		}
		lib.Concepts = append(lib.Concepts, storage.TranslationConcept{
			ID:            uniqueConceptID(lib, base),
			SourceWordID:  ensureWord(words, "hu", hu),
			ContextSource: strings.TrimSpace(nc.ReverseContext),
			TargetWordIDs: []string{dutchID},
			Groups:        slices.Clone(groups),
			NextReview:    now,
		})
	}

	return res, nil
}

// matches is true als het concept van srcLang naar targetLang gaat met deze brontekst (hoofdletterongevoelig)
func matches(words map[string]storage.Word, c storage.TranslationConcept, srcLang, srcText, targetLang string) bool {
	src := words[c.SourceWordID]
	if src.Language != srcLang || !strings.EqualFold(strings.TrimSpace(src.Text), strings.TrimSpace(srcText)) {
		return false
	}
	return len(c.TargetWordIDs) > 0 && words[c.TargetWordIDs[0]].Language == targetLang
}

// TargetTexts geeft de geaccepteerde antwoorden van een concept
func TargetTexts(lib *storage.Library, c storage.TranslationConcept) []string {
	var texts []string
	for _, id := range c.TargetWordIDs {
		texts = append(texts, lib.Words[id].Text)
	}
	return texts
}

// ensureWord geeft het ID van een bestaand woord met deze taal en tekst, of maakt een nieuw woord aan
func ensureWord(words map[string]storage.Word, lang, text string) string {
	var found []string
	for id, w := range words {
		if w.Language == lang && strings.EqualFold(strings.TrimSpace(w.Text), text) {
			found = append(found, id)
		}
	}
	if len(found) > 0 {
		slices.Sort(found) // vaste keuze als hetzelfde woord dubbel bestaat (bijv. "zij")
		return found[0]
	}

	id := uniqueID("w_"+lang+"_"+safeID(text), func(id string) bool { _, taken := words[id]; return taken })
	words[id] = storage.Word{ID: id, Language: lang, Text: text}
	return id
}

// exampleLang: voorbeeldzinnen zijn in de taal die je leert
func exampleLang(example string) string {
	if example == "" {
		return ""
	}
	return "hu"
}

func safeID(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "_")
}

func uniqueConceptID(lib *storage.Library, base string) string {
	return uniqueID(base, func(id string) bool { return lib.Index(id) >= 0 })
}

// uniqueID voegt _2, _3, ... toe zolang het ID al bezet is
func uniqueID(base string, taken func(string) bool) string {
	id := base
	for n := 2; taken(id); n++ {
		id = fmt.Sprintf("%s_%d", base, n)
	}
	return id
}
