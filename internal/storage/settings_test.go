package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "instellingen.json")
	s, err := LoadSettings(path)
	if err != nil || s.NewPerDay != DefaultNewPerDay {
		t.Fatalf("ontbrekend bestand: verwacht standaard %d, kreeg %+v (%v)", DefaultNewPerDay, s, err)
	}
	if err := SaveSettings(path, Settings{NewPerDay: 0}); err != nil {
		t.Fatal(err)
	}
	if s, _ := LoadSettings(path); s.NewPerDay != 0 {
		t.Errorf("0 (geen limiet) moet bewaard blijven, kreeg %d", s.NewPerDay)
	}
	if err := SaveSettings(path, Settings{NewPerDay: -1}); err == nil {
		t.Error("negatieve waarde moet geweigerd worden")
	}
	os.WriteFile(path, []byte(`{"new_per_day":`), 0644)
	if _, err := LoadSettings(path); err == nil {
		t.Error("corrupt bestand moet een fout geven")
	}
}

func TestFirstReviewIsStoredInProgress(t *testing.T) {
	dir := t.TempDir()
	words := writeTestFile(t, dir, "woorden.json", testWords)
	progress := filepath.Join(dir, "voortgang.json")

	_, concepts, _ := LoadData(words, progress)
	if !concepts[0].FirstReview.IsZero() {
		t.Fatal("een nieuw concept heeft nog geen eerste antwoord")
	}
	first := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	concepts[0].FirstReview = first
	if err := SaveProgress(progress, concepts); err != nil {
		t.Fatal(err)
	}
	_, reloaded, _ := LoadData(words, progress)
	if !reloaded[0].FirstReview.Equal(first) {
		t.Errorf("eerste antwoord niet bewaard: %v", reloaded[0].FirstReview)
	}
}
