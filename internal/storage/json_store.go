package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// InputData representeert de structuur van woorden.json
type InputData struct {
	Words    []Word               `json:"words"`
	Concepts []TranslationConcept `json:"concepts"`
}

// ProgressState is de structuur waarin we de voortgang per concept opslaan in voortgang.json
type ProgressState struct {
	StepIndex   int       `json:"step_index"`
	NextReview  time.Time `json:"next_review"`
	FirstReview time.Time `json:"first_review,omitzero"` // leeg = nog nooit gevraagd (een nieuw woord)
	LastReview  time.Time `json:"last_review,omitzero"`  // laatste echte antwoord
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
		return nil, nil, fmt.Errorf("%s is corrupt: %w", wordsFile, err)
	}

	// Zet de words slice om naar een map voor snelle ID-lookups in de frontend
	wordsPool := make(map[string]Word)
	for _, w := range input.Words {
		wordsPool[w.ID] = w
	}

	// 2. Lees voortgang.json. Ontbreekt het bestand, dan starten we zonder voortgang.
	// Is het onleesbaar of corrupt, dan stoppen we: anders overschrijft de volgende save alle voortgang.
	progressMap := make(map[string]ProgressState)
	progressData, err := os.ReadFile(progressFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// eerste keer opstarten
	case err != nil:
		return nil, nil, err
	default:
		if err := json.Unmarshal(progressData, &progressMap); err != nil {
			return nil, nil, fmt.Errorf("%s is corrupt: %w", progressFile, err)
		}
	}

	// 3. Vlecht de voortgang in de concepten. Bestaat er geen voortgang? Dan starten we op NU.
	for i := range input.Concepts {
		conceptID := input.Concepts[i].ID
		if state, exists := progressMap[conceptID]; exists {
			input.Concepts[i].StepIndex = state.StepIndex
			input.Concepts[i].NextReview = state.NextReview
			input.Concepts[i].FirstReview = state.FirstReview
			input.Concepts[i].LastReview = state.LastReview
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
			StepIndex:   c.StepIndex,
			NextReview:  c.NextReview,
			FirstReview: c.FirstReview,
			LastReview:  c.LastReview,
		}
	}

	// Marshal naar leesbare JSON met indentatie
	data, err := json.MarshalIndent(progressMap, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(progressFile, data)
}

// SaveWords schrijft de geüpdatete woorden-pool en concepten terug naar woorden.json
func SaveWords(wordsFile string, wordsPool map[string]Word, concepts []TranslationConcept) error {
	// Bouw de words-pool map weer om naar een platte slice voor JSON
	var wordsSlice []Word
	for _, w := range wordsPool {
		wordsSlice = append(wordsSlice, w)
	}
	// Vaste volgorde, zodat woorden.json niet bij elke save door elkaar wordt gegooid
	slices.SortFunc(wordsSlice, func(a, b Word) int { return strings.Compare(a.ID, b.ID) })

	// De voortgangsvelden van een concept hebben json:"-", dus woorden.json blijft schoon
	data := InputData{
		Words:    wordsSlice,
		Concepts: concepts,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(wordsFile, jsonData)
}

// writeFileAtomic schrijft eerst naar een tijdelijk bestand in dezelfde map en hernoemt dat daarna.
// Een crash halverwege laat zo altijd het oude of het nieuwe bestand achter, nooit een half bestand.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op na een geslaagde rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
