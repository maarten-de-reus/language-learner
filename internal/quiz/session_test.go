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

func TestWrongAnswerIsRetriedAtEndOfRun(t *testing.T) {
	s, _ := newTestSession(t)
	first := s.Current().ID

	fb, _ := s.Answer("fout")
	if !fb.WillRetry || s.Current().RunSize != 3 {
		t.Fatalf("verwacht dat het concept achteraan de run komt, kreeg %+v", fb)
	}
	s.Next()
	if s.Current().ID == first || s.Current().Retry {
		t.Error("de herhaling mag niet direct na de fout komen")
	}
	s.Answer("víz")
	s.Next()

	card := s.Current()
	if card.ID != first || !card.Retry || card.Position != 3 {
		t.Errorf("verwacht de herhaling van %s op positie 3, kreeg %+v", first, card)
	}
}

func TestRetryDoesNotChangeSchedule(t *testing.T) {
	s, _ := newTestSession(t)
	s.Answer("fout") // koffie: reset naar stap 0, morgen weer
	s.Next()
	s.Answer("víz")
	s.Next()

	before := s.concepts[s.run[s.pos].concept]
	fb, err := s.Answer("kávé")
	if err != nil {
		t.Fatal(err)
	}
	after := s.concepts[s.run[s.pos].concept]
	if !fb.Correct || !fb.Retry || fb.WillRetry {
		t.Errorf("verwacht een goede herhaling zonder nieuwe herhaling, kreeg %+v", fb)
	}
	if after.StepIndex != 0 || !after.NextReview.Equal(before.NextReview) {
		t.Errorf("herhaling mag de planning niet veranderen: voor %+v, na %+v", before, after)
	}

	r := s.Results()[0]
	if r.Correct || r.Retries != 1 || !r.RetryCorrect {
		t.Errorf("verwacht: eerste poging fout, 1 herhaling, alsnog goed; kreeg %+v", r)
	}
	if len(s.Results()) != 2 {
		t.Errorf("herhalingen mogen geen extra resultaten toevoegen, kreeg %d", len(s.Results()))
	}
}

func TestRetriesAreCapped(t *testing.T) {
	s, _ := newTestSession(t)
	answers := 0
	for !s.Done() {
		s.Answer("fout")
		s.Next()
		answers++
	}
	// 2 concepten x (1 poging + MaxRetries herhalingen)
	if want := 2 * (1 + MaxRetries); answers != want {
		t.Errorf("verwacht %d vragen, kreeg %d", want, answers)
	}
	for _, r := range s.Results() {
		if r.Retries != MaxRetries || r.RetryCorrect {
			t.Errorf("verwacht %d mislukte herhalingen, kreeg %+v", MaxRetries, r)
		}
	}
}

func TestCorrectAnswerIsNotRetried(t *testing.T) {
	s, _ := newTestSession(t)
	if fb, _ := s.Answer("kávé"); fb.WillRetry {
		t.Error("een goed antwoord mag niet herhaald worden")
	}
	if s.Current().RunSize != 2 {
		t.Errorf("run mag niet groeien, kreeg %d", s.Current().RunSize)
	}
}
