package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReviewLogAppendAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "antwoorden.jsonl")
	log := NewReviewLog(path)
	now := time.Now().Truncate(time.Second)

	reviews := []Review{
		{Time: now, ConceptID: "c_1", Input: "kave", Correct: false, AccentOnly: true, StepBefore: 2, StepAfter: 0, NextReview: now.Add(24 * time.Hour)},
		{Time: now.Add(time.Minute), ConceptID: "c_1", Input: "kávé", Correct: true, Retry: true},
	}
	for _, r := range reviews {
		if err := log.Append(r); err != nil {
			t.Fatal(err)
		}
	}

	got, skipped, err := ReadReviews(path)
	if err != nil || skipped != 0 {
		t.Fatalf("lezen: %v, %d overgeslagen", err, skipped)
	}
	if len(got) != 2 || got[0].ConceptID != "c_1" || !got[0].AccentOnly || !got[1].Retry || !got[0].Time.Equal(now) {
		t.Errorf("teruggelezen: %+v", got)
	}
}

func TestReadReviewsMissingFile(t *testing.T) {
	got, skipped, err := ReadReviews(filepath.Join(t.TempDir(), "bestaat-niet.jsonl"))
	if err != nil || skipped != 0 || len(got) != 0 {
		t.Errorf("ontbrekend bestand moet een lege geschiedenis zijn, kreeg %v %d %v", got, skipped, err)
	}
}

func TestReadReviewsSkipsTornLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "antwoorden.jsonl")
	NewReviewLog(path).Append(Review{ConceptID: "c_1", Correct: true})
	// Een crash midden in het schrijven laat een halve regel achter
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(`{"time":"2026-10-20T21:00:00Z","concept_id":"c_2","inp`)
	f.Close()
	NewReviewLog(path).Append(Review{ConceptID: "c_3", Correct: false})

	got, skipped, err := ReadReviews(path)
	if err != nil {
		t.Fatal(err)
	}
	// Alleen de halve regel gaat verloren; het antwoord erna staat op een eigen regel
	if skipped != 1 || len(got) != 2 || got[0].ConceptID != "c_1" || got[1].ConceptID != "c_3" {
		t.Errorf("verwacht c_1 en c_3 goed en 1 overgeslagen regel, kreeg %d overgeslagen: %+v", skipped, got)
	}
}
