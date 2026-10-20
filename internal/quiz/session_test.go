package quiz

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"hungarian-language-learner/internal/engine"
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

// correctAnswer geeft het goede antwoord op de huidige vraag; de volgorde in een run is willekeurig
func correctAnswer(s *Session) string {
	c := *s.conceptOf(s.run[s.pos])
	return s.lib.Words[c.TargetWordIDs[0]].Text
}

func newTestSession(t *testing.T) (*Session, string) {
	t.Helper()
	words, concepts := testData()
	path := filepath.Join(t.TempDir(), "voortgang.json")
	return New(&storage.Library{Words: words, Concepts: concepts}, path, 10), path
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
	want := s.Current().Step + 1

	fb, err := s.Answer(correctAnswer(s))
	if err != nil {
		t.Fatal(err)
	}
	if !fb.Correct || fb.Step != want {
		t.Errorf("verwacht correct en stap %d, kreeg %+v", want, fb)
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
		if c.ID == id && c.StepIndex != want {
			t.Errorf("opgeslagen stap voor %s: %d, verwacht %d", id, c.StepIndex, want)
		}
	}
}

func TestAnswerTwiceIsIgnored(t *testing.T) {
	s, _ := newTestSession(t)
	answer := correctAnswer(s)
	s.Answer("fout")
	if fb, _ := s.Answer(answer); fb != nil {
		t.Error("tweede antwoord op hetzelfde concept mag niet nagekeken worden")
	}
	if len(s.Results()) != 1 || s.Results()[0].Correct {
		t.Errorf("verwacht één fout resultaat, kreeg %+v", s.Results())
	}
}

func TestAccentOnlyIsWrongWithHint(t *testing.T) {
	s, _ := newTestSession(t)
	fb, _ := s.Answer(engine.Fold(correctAnswer(s))) // kávé -> kave, víz -> viz
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
	s.Answer(correctAnswer(s))
	s.Next()

	card := s.Current()
	if card.ID != first || !card.Retry || card.Position != 3 {
		t.Errorf("verwacht de herhaling van %s op positie 3, kreeg %+v", first, card)
	}
}

func TestRetryDoesNotChangeSchedule(t *testing.T) {
	s, _ := newTestSession(t)
	s.Answer("fout") // eerste woord: reset naar stap 0, morgen weer
	s.Next()
	s.Answer(correctAnswer(s))
	s.Next()

	before := *s.conceptOf(s.run[s.pos])
	fb, err := s.Answer(correctAnswer(s))
	if err != nil {
		t.Fatal(err)
	}
	after := *s.conceptOf(s.run[s.pos])
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
	if fb, _ := s.Answer(correctAnswer(s)); fb.WillRetry {
		t.Error("een goed antwoord mag niet herhaald worden")
	}
	if s.Current().RunSize != 2 {
		t.Errorf("run mag niet groeien, kreeg %d", s.Current().RunSize)
	}
}

// időData: idő met twee betekenissen (weer / tijd) en een Hongaarse voorbeeldzin
func időData(t *testing.T) *Session {
	t.Helper()
	words := map[string]storage.Word{
		"w_hu_idő":  {ID: "w_hu_idő", Language: "hu", Text: "idő"},
		"w_nl_weer": {ID: "w_nl_weer", Language: "nl", Text: "weer"},
		"w_nl_tijd": {ID: "w_nl_tijd", Language: "nl", Text: "tijd"},
	}
	past := time.Now().Add(-time.Hour)
	concepts := []storage.TranslationConcept{
		{ID: "c_weer", SourceWordID: "w_hu_idő", ContextSource: "buiten", TargetWordIDs: []string{"w_nl_weer"},
			Example: "Milyen lesz az idő holnap?", ExampleLang: "hu", StepIndex: 2, NextReview: past},
		{ID: "c_tijd", SourceWordID: "w_hu_idő", ContextSource: "klok", TargetWordIDs: []string{"w_nl_tijd"},
			NextReview: time.Now().Add(48 * time.Hour)},
		{ID: "c_nl_weer", SourceWordID: "w_nl_weer", TargetWordIDs: []string{"w_hu_idő"},
			Example: "Milyen lesz az idő holnap?", ExampleLang: "hu", StepIndex: 3, NextReview: past},
	}
	return New(&storage.Library{Words: words, Concepts: concepts}, filepath.Join(t.TempDir(), "voortgang.json"), 10)
}

func TestOtherMeaningIsWrongWithExplanation(t *testing.T) {
	s := időData(t)
	if s.Current().ID != "c_weer" {
		t.Fatalf("verwacht c_weer eerst, kreeg %s", s.Current().ID)
	}
	fb, _ := s.Answer("Tijd")
	if fb.Correct || fb.Step != 0 {
		t.Errorf("de andere betekenis moet als fout tellen, kreeg %+v", fb)
	}
	if fb.OtherMeaning == nil || fb.OtherMeaning.Answer != "tijd" || fb.OtherMeaning.Context != "klok" {
		t.Errorf("verwacht uitleg over de betekenis 'tijd (klok)', kreeg %+v", fb.OtherMeaning)
	}
}

func TestRandomWrongAnswerHasNoOtherMeaning(t *testing.T) {
	s := időData(t)
	if fb, _ := s.Answer("zon"); fb.OtherMeaning != nil {
		t.Errorf("geen andere betekenis verwacht, kreeg %+v", fb.OtherMeaning)
	}
}

func TestExampleShownBeforeOnlyWhenItDoesNotGiveAwayTheAnswer(t *testing.T) {
	s := időData(t)

	// HU->NL: Hongaarse zin bij een Hongaarse vraag mag vooraf
	if got := s.Current().Example; got != "Milyen lesz az idő holnap?" {
		t.Errorf("HU->NL: verwacht de voorbeeldzin op de kaart, kreeg %q", got)
	}
	s.Answer("weer")
	s.Next()

	// Beide richtingen van idő/weer komen nooit in dezelfde run; NL->HU volgt in de volgende run
	if !s.Done() || !s.NewRun() {
		t.Fatal("verwacht dat de run klaar is en een nieuwe run start")
	}

	// NL->HU: de Hongaarse zin bevat het antwoord, dus pas na het antwoorden
	if card := s.Current(); card.ID != "c_nl_weer" || card.Example != "" {
		t.Errorf("NL->HU: voorbeeldzin mag niet vooraf getoond worden, kreeg %+v", card)
	}
	if fb, _ := s.Answer("idő"); fb.Example != "Milyen lesz az idő holnap?" {
		t.Errorf("NL->HU: verwacht de voorbeeldzin in de feedback, kreeg %q", fb.Example)
	}
}

func TestAnswersAreLogged(t *testing.T) {
	s, path := newTestSession(t)
	logPath := filepath.Join(filepath.Dir(path), "antwoorden.jsonl")
	s.LogReviewsTo(storage.NewReviewLog(logPath))

	first := s.Current().ID
	stepBefore := s.Current().Step
	answer := correctAnswer(s)
	s.Answer("fout") // komt terug als oefenherhaling
	s.Next()
	s.Answer(correctAnswer(s))
	s.Next()
	s.Answer(answer) // de herhaling, nu goed

	reviews, skipped, err := storage.ReadReviews(logPath)
	if err != nil || skipped != 0 {
		t.Fatalf("lezen: %v (%d overgeslagen)", err, skipped)
	}
	if len(reviews) != 3 {
		t.Fatalf("verwacht 3 antwoorden in het log, kreeg %d", len(reviews))
	}
	wrong, retry := reviews[0], reviews[2]
	if wrong.ConceptID != first || wrong.Correct || wrong.Retry || wrong.Input != "fout" || wrong.StepBefore != stepBefore || wrong.StepAfter != 0 {
		t.Errorf("eerste antwoord: %+v", wrong)
	}
	if retry.ConceptID != first || !retry.Correct || !retry.Retry || retry.StepBefore != 0 || retry.StepAfter != 0 {
		t.Errorf("oefenherhaling moet als retry gelogd worden zonder stapwijziging: %+v", retry)
	}
}

func TestBrokenLogDoesNotBlockPractice(t *testing.T) {
	s, path := newTestSession(t)
	s.LogReviewsTo(storage.NewReviewLog(filepath.Join(filepath.Dir(path), "bestaat-niet", "antwoorden.jsonl")))

	fb, err := s.Answer(correctAnswer(s))
	if err == nil {
		t.Error("een mislukt log moet als fout gemeld worden")
	}
	if fb == nil || !fb.Correct {
		t.Fatalf("het antwoord moet gewoon nagekeken zijn, kreeg %+v", fb)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("de voortgang moet wel opgeslagen zijn: %v", err)
	}
}

func TestForgetRemovesDeletedConceptsFromRun(t *testing.T) {
	s, _ := newTestSession(t)
	first := s.Current().ID
	s.Answer("fout") // first komt ook nog terug als oefenherhaling
	s.Next()
	second := s.Current().ID

	// Het eerder beantwoorde concept (en zijn herhaling) wordt verwijderd: de huidige vraag blijft
	s.Forget(first)
	if s.Current() == nil || s.Current().ID != second || s.Current().Position != 1 || s.Current().RunSize != 1 {
		t.Fatalf("verwacht %s als enige vraag op positie 1, kreeg %+v", second, s.Current())
	}

	// Het huidige concept wordt verwijderd: de run is dan klaar
	s.Forget(second)
	if !s.Done() || s.Feedback() != nil {
		t.Errorf("verwacht een afgeronde run zonder feedback, kreeg %+v", s.Current())
	}
	if len(s.Results()) != 1 {
		t.Errorf("resultaten van al gegeven antwoorden blijven bewaard, kreeg %d", len(s.Results()))
	}
}
