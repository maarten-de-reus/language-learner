package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"hungarian-language-learner/internal/storage"
)

var (
	ErrDuplicate        = errors.New("dit woord met deze context bestaat al in de database")
	ErrReverseAmbiguous = errors.New("er bestaat al een HU->NL concept met dit woord en deze hint; kies een andere hint")
)

// newConcept is de invoer voor een nieuw NL->HU / HU->NL paar
type newConcept struct {
	Dutch     string
	Context   string   // hint bij NL->HU, bijv. "Meubel"
	Hungarian []string // geaccepteerde Hongaarse vertalingen; de eerste is de bron van HU->NL
	Example   string   // optionele Hongaarse voorbeeldzin, voor beide richtingen

	// HU->NL: óf samenvoegen met een bestaand concept met dezelfde betekenis (bijv. ő -> hij/zij),
	// óf een nieuw concept met een eigen hint die de betekenis aanwijst (bijv. idő -> weer / tijd)
	MergeReverseInto string            // ID van bestaand HU->NL concept; leeg = nieuw concept
	ReverseContext   string            // hint voor het nieuwe HU->NL concept
	ExistingHints    map[string]string // hints voor bestaande HU->NL concepten die er nog geen hadden
}

// addResult beschrijft wat addConceptPair heeft gedaan, voor de melding aan de gebruiker
type addResult struct {
	ReverseSource string // Hongaars woord dat de bron is van het HU->NL concept
	ReverseMerged bool   // true als het NL-woord aan een bestaand HU->NL concept is toegevoegd
}

// parseInput schoont de invoer op; meerdere Hongaarse vertalingen worden gescheiden door komma's
func parseInput(dutch, context, hungarianInput string) (newConcept, error) {
	nc := newConcept{Dutch: strings.TrimSpace(dutch), Context: strings.TrimSpace(context)}
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

// isDuplicate herkent een bestaand NL->HU concept op tekst en context, niet op ID:
// ID's zijn alleen labels (oude data heeft genummerde ID's zoals c_1 en w_hu_11)
func isDuplicate(words map[string]storage.Word, concepts []storage.TranslationConcept, nc newConcept) bool {
	return slices.ContainsFunc(concepts, func(c storage.TranslationConcept) bool {
		return matches(words, c, "nl", nc.Dutch, "hu") && strings.EqualFold(strings.TrimSpace(c.ContextSource), nc.Context)
	})
}

// reverseCandidates geeft de bestaande HU->NL concepten met dit Hongaarse woord als bron
func reverseCandidates(words map[string]storage.Word, concepts []storage.TranslationConcept, hungarian string) []int {
	var idx []int
	for i, c := range concepts {
		if matches(words, c, "hu", hungarian, "nl") {
			idx = append(idx, i)
		}
	}
	return idx
}

// reverseHintTaken is true als een ander HU->NL concept met dit woord al deze hint heeft
func reverseHintTaken(words map[string]storage.Word, concepts []storage.TranslationConcept, nc newConcept, hint string) bool {
	for _, i := range reverseCandidates(words, concepts, nc.Hungarian[0]) {
		existing := concepts[i].ContextSource
		if h, ok := nc.ExistingHints[concepts[i].ID]; ok {
			existing = h
		}
		if strings.EqualFold(strings.TrimSpace(existing), strings.TrimSpace(hint)) {
			return true
		}
	}
	return false
}

// addConceptPair voegt het NL->HU concept toe en maakt of verrijkt het HU->NL concept.
// Alle controles gebeuren vóór er iets aangepast wordt.
func addConceptPair(words map[string]storage.Word, concepts []storage.TranslationConcept, nc newConcept) ([]storage.TranslationConcept, addResult, error) {
	res := addResult{ReverseSource: nc.Hungarian[0], ReverseMerged: nc.MergeReverseInto != ""}

	// --- CONTROLES ---
	if isDuplicate(words, concepts, nc) {
		return concepts, res, ErrDuplicate
	}
	mergeIdx := -1
	if res.ReverseMerged {
		mergeIdx = indexOf(concepts, nc.MergeReverseInto)
		if mergeIdx < 0 || !matches(words, concepts[mergeIdx], "hu", nc.Hungarian[0], "nl") {
			return concepts, res, fmt.Errorf("HU->NL concept %q hoort niet bij %q", nc.MergeReverseInto, nc.Hungarian[0])
		}
	} else if reverseHintTaken(words, concepts, nc, nc.ReverseContext) {
		return concepts, res, ErrReverseAmbiguous
	}

	// --- VERWERK DE WOORDEN IN DE POOL (bestaande woorden worden hergebruikt) ---
	dutchID := ensureWord(words, "nl", nc.Dutch)
	var huIDs []string
	for _, hu := range nc.Hungarian {
		if id := ensureWord(words, "hu", hu); !slices.Contains(huIDs, id) {
			huIDs = append(huIDs, id)
		}
	}

	// --- CONCEPT 1: Nederlands -> Hongaars ---
	base := "c_nl_hu_" + safeID(nc.Dutch)
	if nc.Context != "" {
		base += "_" + safeID(nc.Context)
	}
	concepts = append(concepts, storage.TranslationConcept{
		ID:            uniqueConceptID(concepts, base),
		SourceWordID:  dutchID,
		ContextSource: nc.Context,
		TargetWordIDs: huIDs,
		Example:       nc.Example,
		ExampleLang:   exampleLang(nc.Example),
	})

	// --- CONCEPT 2: Hongaars -> Nederlands ---
	if mergeIdx >= 0 {
		merged := &concepts[mergeIdx]
		if !slices.Contains(merged.TargetWordIDs, dutchID) {
			merged.TargetWordIDs = append(merged.TargetWordIDs, dutchID)
		}
		if merged.Example == "" && nc.Example != "" {
			merged.Example, merged.ExampleLang = nc.Example, exampleLang(nc.Example)
		}
		return concepts, res, nil
	}
	for id, hint := range nc.ExistingHints {
		if i := indexOf(concepts, id); i >= 0 {
			concepts[i].ContextSource = strings.TrimSpace(hint)
		}
	}
	base = "c_hu_nl_" + safeID(nc.Hungarian[0])
	if nc.ReverseContext != "" {
		base += "_" + safeID(nc.ReverseContext)
	}
	concepts = append(concepts, storage.TranslationConcept{
		ID:            uniqueConceptID(concepts, base),
		SourceWordID:  huIDs[0],
		ContextSource: strings.TrimSpace(nc.ReverseContext),
		TargetWordIDs: []string{dutchID},
		Example:       nc.Example,
		ExampleLang:   exampleLang(nc.Example),
	})
	return concepts, res, nil
}

// matches is true als het concept van srcLang naar targetLang gaat met deze brontekst (hoofdletterongevoelig)
func matches(words map[string]storage.Word, c storage.TranslationConcept, srcLang, srcText, targetLang string) bool {
	src := words[c.SourceWordID]
	if src.Language != srcLang || !strings.EqualFold(strings.TrimSpace(src.Text), strings.TrimSpace(srcText)) {
		return false
	}
	return len(c.TargetWordIDs) > 0 && words[c.TargetWordIDs[0]].Language == targetLang
}

// targetTexts geeft de geaccepteerde antwoorden van een concept, voor meldingen
func targetTexts(words map[string]storage.Word, c storage.TranslationConcept) string {
	var texts []string
	for _, id := range c.TargetWordIDs {
		texts = append(texts, words[id].Text)
	}
	return strings.Join(texts, ", ")
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

func uniqueConceptID(concepts []storage.TranslationConcept, base string) string {
	return uniqueID(base, func(id string) bool { return indexOf(concepts, id) >= 0 })
}

// uniqueID voegt _2, _3, ... toe zolang het ID al bezet is
func uniqueID(base string, taken func(string) bool) string {
	id := base
	for n := 2; taken(id); n++ {
		id = fmt.Sprintf("%s_%d", base, n)
	}
	return id
}

func indexOf(concepts []storage.TranslationConcept, id string) int {
	return slices.IndexFunc(concepts, func(c storage.TranslationConcept) bool { return c.ID == id })
}
