package quiz

import (
	"path/filepath"
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

func testData() (map[string]storage.Word, []storage.TranslationConcept) {
	words := map[string]storage.Word{
		"w_nl_koffie": {ID: "w_nl_koffie", Language: "nl", Text: "koffie"},
		"w_hu_kave":   {ID: "w_hu_kave", Language: "hu", Text: "kávé"},
		"w_nl_water":  {ID: "w_nl_water", Language: "nl", Text: "water"},
		"w_hu_viz":    {ID: "w_hu_viz", Language: "hu", Text: "víz"},
	}
	past := time.Now().Add(-time.Hour)
	concepts := []storage.TranslationConcept{
		{ID: "c_koffie", SourceWordID: "w_nl_koffie", TargetWordIDs: []string{"w_hu_kave"}, StepIndex: 1, NextReview: past},
		{ID: "c_water", SourceWordID: "w_nl_water", TargetWordIDs: []string{"w_hu_viz"}, StepIndex: 2, NextReview: past},
		{ID: "c_later", SourceWordID: "w_nl_water", TargetWordIDs: []string{"w_hu_viz"}, NextReview: time.Now().Add(48 * time.Hour)},
	}
	return words, concepts
}

func newTestSession(t *testing.T) (*Session, string) {
	t.Helper()
	words, concepts := testData()
	path := filepath.Join(t.TempDir(), "voortgang.json")
	return New(words, concepts, path, 10), path
}

func TestRunContainsOnlyDueConcepts(t *testing.T) {
	s, _ := newTestSession(t)
	card := s.Current()
	if card == nil || card.RunSize != 2 {
		t.Fatalf("verwacht een run van 2 concepten, kreeg %+v", card)
	}
	if card.From != "nl" || card.To != "hu" {
		t.Errorf("richting: %s -> %s", card.From, card.To)
	}
}

func TestAnswerUpdatesAndPersistsProgress(t *testing.T) {
	s, path := newTestSession(t)
	id := s.Current().ID

	fb, err := s.Answer("kávé")
	if err != nil {
		t.Fatal(err)
	}
	if !fb.Correct || fb.Step != 2 {
		t.Errorf("verwacht correct en stap 2, kreeg %+v", fb)
	}

	// De voortgang moet echt in het bestand staan (dit ging eerder mis in de web-frontend)
	words, concepts := testData()
	if err := storage.SaveWords(filepath.Join(filepath.Dir(path), "woorden.json"), words, concepts); err != nil {
		t.Fatal(err)
	}
	_, reloaded, err := storage.LoadData(filepath.Join(filepath.Dir(path), "woorden.json"), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range reloaded {
		if c.ID == id && c.StepIndex != 2 {
			t.Errorf("opgeslagen stap voor %s: %d, verwacht 2", id, c.StepIndex)
		}
	}
}

func TestAnswerTwiceIsIgnored(t *testing.T) {
	s, _ := newTestSession(t)
	s.Answer("fout")
	if fb, _ := s.Answer("kávé"); fb != nil {
		t.Error("tweede antwoord op hetzelfde concept mag niet nagekeken worden")
	}
	if len(s.Results()) != 1 || s.Results()[0].Correct {
		t.Errorf("verwacht één fout resultaat, kreeg %+v", s.Results())
	}
}

func TestAccentOnlyIsWrongWithHint(t *testing.T) {
	s, _ := newTestSession(t)
	fb, _ := s.Answer("kave")
	if fb.Correct || !fb.AccentOnly || fb.Step != 0 {
		t.Errorf("verwacht fout met accent-hint en reset naar stap 0, kreeg %+v", fb)
	}
}

func TestNextAndNewRunGuards(t *testing.T) {
	s, _ := newTestSession(t)
	if s.Next() {
		t.Error("Next zonder antwoord moet niets doen")
	}
	if s.NewRun() {
		t.Error("NewRun tijdens een lopende run moet niets doen")
	}

	for !s.Done() {
		s.Answer("x")
		s.Next()
	}
	if len(s.Results()) != 2 {
		t.Fatalf("verwacht 2 resultaten, kreeg %d", len(s.Results()))
	}

	// Fout beantwoord = morgen weer due, dus de nieuwe run is leeg
	if !s.NewRun() || !s.Done() || len(s.Results()) != 0 {
		t.Error("verwacht een lege nieuwe run met gewiste resultaten")
	}
}

func TestStats(t *testing.T) {
	s, _ := newTestSession(t)
	st := s.Stats()
	if st.Total != 3 || st.Due != 2 {
		t.Errorf("verwacht 3 totaal en 2 due, kreeg %d / %d", st.Total, st.Due)
	}
	if st.Steps[0] != 1 || st.Steps[1] != 1 || st.Steps[2] != 1 {
		t.Errorf("stapverdeling: %v", st.Steps)
	}
	if st.NextDue == nil {
		t.Error("verwacht een volgende herhaling")
	}
}
