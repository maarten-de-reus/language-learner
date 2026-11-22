package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// RuleProgress is de planning van één grammaticaregel in de herhalingen (Oefenen -> Regels),
// net als de voortgang van een woord: een stap uit engine.AnkiIntervals en de volgende datum
type RuleProgress struct {
	Step        int       `json:"step"`
	NextReview  time.Time `json:"next_review"`
	FirstReview time.Time `json:"first_review"`
	LastReview  time.Time `json:"last_review"`
}

// LoadGrammarProgress leest regels-voortgang.json; een ontbrekend bestand is geen voortgang
func LoadGrammarProgress(path string) (map[string]RuleProgress, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]RuleProgress{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]RuleProgress{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("%s is corrupt: %w", path, err)
	}
	return out, nil
}

// SaveGrammarProgress schrijft de planning van de regels atomair weg
func SaveGrammarProgress(path string, p map[string]RuleProgress) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data)
}
