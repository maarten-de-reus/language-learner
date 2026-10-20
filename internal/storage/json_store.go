package storage

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// InputData representeert de structuur van woorden.json
type InputData struct {
	Words    []Word               `json:"words"`
	Concepts []TranslationConcept `json:"concepts"`
}

// ProgressState is de structuur waarin we de voortgang per concept opslaan in voortgang.json
type ProgressState struct {
	StepIndex  int       `json:"step_index"`
	NextReview time.Time `json:"next_review"`
}

// LoadData laadt de woorden in en vlecht de opgeslagen voortgang er doorheen
func LoadData(wordsFile, progressFile string) (map[string]Word, []TranslationConcept, error) {
	// 1. Lees woorden.json
	wordsData, err := os.ReadFile(wordsFile)
	if err != nil {
		return nil, nil, err
	}

	var input InputData
	if err := json.Unmarshal(wordsData, &input); err != nil {
		return nil, nil, err
	}

	// Zet de words slice om naar een map voor snelle ID-lookups in de frontend
	wordsPool := make(map[string]Word)
	for _, w := range input.Words {
		wordsPool[w.ID] = w
	}

	// 2. Probeer voortgang.json te lezen (als deze bestaat)
	progressMap := make(map[string]ProgressState)
	progressData, err := os.ReadFile(progressFile)
	if err == nil {
		_ = json.Unmarshal(progressData, &progressMap)
	}

	// 3. Vlecht de voortgang in de concepten. Bestaat er geen voortgang? Dan starten we op NU.
	for i := range input.Concepts {
		conceptID := input.Concepts[i].ID
		if state, exists := progressMap[conceptID]; exists {
			input.Concepts[i].StepIndex = state.StepIndex
			input.Concepts[i].NextReview = state.NextReview
		} else {
			input.Concepts[i].StepIndex = 0
			input.Concepts[i].NextReview = time.Now()
		}
	}

	return wordsPool, input.Concepts, nil
}

// SaveProgress trekt de voortgang uit de runtime-concepten en slaat dit op in voortgang.json
func SaveProgress(progressFile string, concepts []TranslationConcept) error {
	if progressFile == "" {
		return errors.New("progress file path cannot be empty")
	}

	progressMap := make(map[string]ProgressState)

	// Extraheer alleen de statistieken die we willen opslaan
	for _, c := range concepts {
		progressMap[c.ID] = ProgressState{
			StepIndex:  c.StepIndex,
			NextReview: c.NextReview,
		}
	}

	// Marshal naar leesbare JSON met indentatie
	data, err := json.MarshalIndent(progressMap, "", "  ")
	if err != nil {
		return err
	}

	// Schrijf (of overschrijf) het bestand met de juiste Unix permissies
	return os.WriteFile(progressFile, data, 0644)
}

// SaveWords schrijft de geüpdatete woorden-pool en concepten terug naar woorden.json
func SaveWords(wordsFile string, wordsPool map[string]Word, concepts []TranslationConcept) error {
	// Bouw de words-pool map weer om naar een platte slice voor JSON
	var wordsSlice []Word
	for _, w := range wordsPool {
		wordsSlice = append(wordsSlice, w)
	}

	// Filter de runtime Spaced Repetition data eruit, zodat woorden.json schoon blijft
	cleanConcepts := make([]TranslationConcept, len(concepts))
	for i, c := range concepts {
		cleanConcepts[i] = TranslationConcept{
			ID:            c.ID,
			SourceWordID:  c.SourceWordID,
			ContextSource: c.ContextSource,
			TargetWordIDs: c.TargetWordIDs,
		}
	}

	data := InputData{
		Words:    wordsSlice,
		Concepts: cleanConcepts,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(wordsFile, jsonData, 0644)
}
