package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// DefaultNewPerDay is het standaard aantal nieuwe woorden per dag (zoals in Anki)
const DefaultNewPerDay = 20

// Settings zijn de instellingen die je in de interface aanpast; ze staan in instellingen.json
type Settings struct {
	NewPerDay int `json:"new_per_day"` // 0 = geen limiet
	// GrammarRules: de regels die je oefent (zie grammar.Rules). nil = de standaardkeuze;
	// een lege lijst = bewust alles uit.
	GrammarRules []string `json:"grammar_rules"`
	// GrammarMinStep: Regels oefent alleen met woorden die je minstens op deze stap kent
	// (0 = 1 dag … 6 = beheerst). nil = grammar.DefaultMinStep.
	GrammarMinStep *int `json:"grammar_min_step,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{NewPerDay: DefaultNewPerDay}
}

// Validate controleert of de instellingen bruikbaar zijn
func (s Settings) Validate() error {
	if s.NewPerDay < 0 || s.NewPerDay > 500 {
		return fmt.Errorf("nieuwe woorden per dag moet tussen 0 (geen limiet) en 500 liggen, niet %d", s.NewPerDay)
	}
	if s.GrammarMinStep != nil && (*s.GrammarMinStep < 0 || *s.GrammarMinStep > 6) {
		return fmt.Errorf("het niveau voor Regels moet tussen 0 en 6 liggen, niet %d", *s.GrammarMinStep)
	}
	return nil
}

// LoadSettings leest de instellingen; een ontbrekend bestand geeft de standaardwaarden
func LoadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	s := DefaultSettings()
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("%s is corrupt: %w", path, err)
	}
	return s, s.Validate()
}

// SaveSettings schrijft de instellingen atomair weg
func SaveSettings(path string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data)
}
