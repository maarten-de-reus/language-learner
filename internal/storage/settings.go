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
}

func DefaultSettings() Settings {
	return Settings{NewPerDay: DefaultNewPerDay}
}

// Validate controleert of de instellingen bruikbaar zijn
func (s Settings) Validate() error {
	if s.NewPerDay < 0 || s.NewPerDay > 500 {
		return fmt.Errorf("nieuwe woorden per dag moet tussen 0 (geen limiet) en 500 liggen, niet %d", s.NewPerDay)
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
