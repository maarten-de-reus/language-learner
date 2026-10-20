package main

import (
	"errors"
	"slices"
	"strings"

	"hungarian-language-learner/internal/storage"
)

var ErrDuplicate = errors.New("dit woord met deze context bestaat al in de database")

// addResult beschrijft wat addConceptPair heeft gedaan, voor de melding aan de gebruiker
type addResult struct {
	Hungarian     []string // opgeschoonde Hongaarse synoniemen
	ReverseSource string   // Hongaars woord dat de bron is van het HU->NL concept
	ReverseMerged bool     // true als het NL-woord aan een bestaand HU->NL concept is toegevoegd
}

func safeID(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "_")
}

// addConceptPair voegt een NL->HU concept toe en maakt het bijbehorende HU->NL concept aan.
// Bestaat dat HU->NL concept al (bijv. twee NL-woorden met dezelfde Hongaarse vertaling),
// dan wordt het NL-woord daar als extra geaccepteerd antwoord aan toegevoegd.
func addConceptPair(words map[string]storage.Word, concepts []storage.TranslationConcept, dutch, context, hungarianInput string) ([]storage.TranslationConcept, addResult, error) {
	var res addResult

	dutch = strings.TrimSpace(dutch)
	context = strings.TrimSpace(context)
	if dutch == "" {
		return concepts, res, errors.New("woord mag niet leeg zijn")
	}
	for _, hu := range strings.Split(hungarianInput, ",") {
		if hu = strings.TrimSpace(hu); hu != "" {
			res.Hungarian = append(res.Hungarian, hu)
		}
	}
	if len(res.Hungarian) == 0 {
		return concepts, res, errors.New("vertaling mag niet leeg zijn")
	}

	suffix := ""
	if c := safeID(context); c != "" {
		suffix = "_" + c
	}

	// --- DUPLICATEN CHECK ---
	conceptIDNLHU := "c_nl_hu_" + safeID(dutch) + suffix
	if indexOf(concepts, conceptIDNLHU) >= 0 {
		return concepts, res, ErrDuplicate
	}

	// --- VERWERK DE WOORDEN IN DE POOL ---
	dutchID := "w_nl_" + safeID(dutch)
	words[dutchID] = storage.Word{ID: dutchID, Language: "nl", Text: dutch}

	var huIDs []string
	for _, hu := range res.Hungarian {
		id := "w_hu_" + safeID(hu)
		words[id] = storage.Word{ID: id, Language: "hu", Text: hu}
		if !slices.Contains(huIDs, id) {
			huIDs = append(huIDs, id)
		}
	}

	// --- CONCEPT 1: Nederlands -> Hongaars ---
	concepts = append(concepts, storage.TranslationConcept{
		ID:            conceptIDNLHU,
		SourceWordID:  dutchID,
		ContextSource: context,
		TargetWordIDs: huIDs,
	})

	// --- CONCEPT 2: Hongaars -> Nederlands ---
	// Het eerste Hongaarse woord is de bron voor de omgekeerde richting
	res.ReverseSource = res.Hungarian[0]
	conceptIDHUNL := "c_hu_nl_" + safeID(res.ReverseSource) + suffix
	if i := indexOf(concepts, conceptIDHUNL); i >= 0 {
		if !slices.Contains(concepts[i].TargetWordIDs, dutchID) {
			concepts[i].TargetWordIDs = append(concepts[i].TargetWordIDs, dutchID)
		}
		res.ReverseMerged = true
		return concepts, res, nil
	}
	concepts = append(concepts, storage.TranslationConcept{
		ID:            conceptIDHUNL,
		SourceWordID:  huIDs[0],
		ContextSource: "", // Voor een Hongaar is de NL context vaak overbodig
		TargetWordIDs: []string{dutchID},
	})
	return concepts, res, nil
}

func indexOf(concepts []storage.TranslationConcept, id string) int {
	return slices.IndexFunc(concepts, func(c storage.TranslationConcept) bool { return c.ID == id })
}
