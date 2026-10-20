package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testWords = `{
  "words": [
    {"id": "w_nl_bank", "language": "nl", "text": "bank"},
    {"id": "w_hu_pad", "language": "hu", "text": "pad"}
  ],
  "concepts": [
    {"id": "c_1", "source_word_id": "w_nl_bank", "context_source": "Meubel", "target_word_ids": ["w_hu_pad"]}
  ]
}`

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDataWithoutProgressFile(t *testing.T) {
	dir := t.TempDir()
	words := writeTestFile(t, dir, "woorden.json", testWords)

	_, concepts, err := LoadData(words, filepath.Join(dir, "voortgang.json"))
	if err != nil {
		t.Fatalf("ontbrekende voortgang moet geen fout zijn: %v", err)
	}
	if concepts[0].StepIndex != 0 || time.Since(concepts[0].NextReview) > time.Minute {
		t.Errorf("nieuw concept moet op stap 0 staan en nu due zijn, kreeg %+v", concepts[0])
	}
}

func TestLoadDataCorruptProgressFails(t *testing.T) {
	dir := t.TempDir()
	words := writeTestFile(t, dir, "woorden.json", testWords)
	progress := writeTestFile(t, dir, "voortgang.json", `{"c_1": {"step_index": 3,`)

	if _, _, err := LoadData(words, progress); err == nil {
		t.Fatal("corrupte voortgang moet een fout geven, anders wordt hij bij de volgende save gewist")
	}
}

func TestSaveProgressRoundTrip(t *testing.T) {
	dir := t.TempDir()
	words := writeTestFile(t, dir, "woorden.json", testWords)
	progress := filepath.Join(dir, "voortgang.json")

	_, concepts, err := LoadData(words, progress)
	if err != nil {
		t.Fatal(err)
	}
	concepts[0].StepIndex = 4
	if err := SaveProgress(progress, concepts); err != nil {
		t.Fatal(err)
	}

	_, reloaded, err := LoadData(words, progress)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded[0].StepIndex != 4 {
		t.Errorf("verwacht stap 4 na herladen, kreeg %d", reloaded[0].StepIndex)
	}

	// Geen tijdelijke bestanden achtergelaten
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("verwacht alleen woorden.json en voortgang.json, kreeg %d bestanden", len(entries))
	}
}

func TestSaveWordsSortedByID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "woorden.json")
	pool := map[string]Word{
		"w_c": {ID: "w_c", Language: "hu", Text: "c"},
		"w_a": {ID: "w_a", Language: "nl", Text: "a"},
		"w_b": {ID: "w_b", Language: "hu", Text: "b"},
	}

	if err := SaveWords(path, pool, nil); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var saved InputData
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"w_a", "w_b", "w_c"} {
		if saved.Words[i].ID != want {
			t.Errorf("positie %d: verwacht %s, kreeg %s", i, want, saved.Words[i].ID)
		}
	}
}

func TestSaveWordsKeepsExample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "woorden.json")
	concepts := []TranslationConcept{{ID: "c_1", Example: "Milyen az idő?", ExampleLang: "hu", StepIndex: 3}}

	if err := SaveWords(path, map[string]Word{}, concepts); err != nil {
		t.Fatal(err)
	}
	_, loaded, err := LoadData(path, filepath.Join(dir, "voortgang.json"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded[0].Example != "Milyen az idő?" || loaded[0].ExampleLang != "hu" {
		t.Errorf("voorbeeldzin niet bewaard: %+v", loaded[0])
	}
}
