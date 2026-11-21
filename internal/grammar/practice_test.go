package grammar

import (
	"math/rand/v2"
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

func practiceLib() *storage.Library {
	lib := &storage.Library{Words: map[string]storage.Word{
		"nl_huis": {ID: "nl_huis", Language: "nl", Text: "huis"}, "hu_haz": {ID: "hu_haz", Language: "hu", Text: "ház"},
		"nl_leest": {ID: "nl_leest", Language: "nl", Text: "leest"}, "hu_olvas": {ID: "hu_olvas", Language: "hu", Text: "olvas"},
		"nl_lezen": {ID: "nl_lezen", Language: "nl", Text: "lezen"}, "hu_olvasni": {ID: "hu_olvasni", Language: "hu", Text: "olvasni"},
		"nl_tuin": {ID: "nl_tuin", Language: "nl", Text: "tuin"}, "hu_kert": {ID: "hu_kert", Language: "hu", Text: "kert"},
	}}
	lib.Concepts = []storage.TranslationConcept{
		{ID: "a", SourceWordID: "nl_huis", TargetWordIDs: []string{"hu_haz"}, WordType: "zn", FirstReview: time.Now()},
		{ID: "b", SourceWordID: "hu_haz", TargetWordIDs: []string{"nl_huis"}, WordType: "zn"},
		{ID: "c", SourceWordID: "nl_leest", TargetWordIDs: []string{"hu_olvas"}, WordType: "ww"},
		{ID: "d", SourceWordID: "nl_lezen", TargetWordIDs: []string{"hu_olvasni"}, WordType: "ww"},
		{ID: "e", SourceWordID: "nl_tuin", TargetWordIDs: []string{"hu_kert"}},
	}
	return lib
}

func TestWordsAndNext(t *testing.T) {
	words := Words(practiceLib())
	if len(words) != 2 || words[0].Hungarian != "ház" || !words[0].Known || words[1].Hungarian != "olvas" {
		t.Fatalf("woorden: %+v (verwacht ház en olvas: geen infinitief, geen woord zonder woordsoort, HU->NL niet dubbel)", words)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	seen := map[string]bool{}
	for range 50 {
		ex, ok := Next(words, []string{"ban", "ww-onbep"}, nil, "", rng)
		if !ok {
			t.Fatal("verwacht een opgave")
		}
		seen[ex.Rule+":"+ex.Word] = true
		if ex.Rule == "ww-onbep" && (ex.PersonLabel == "" || ex.Person == int(Sg3)) {
			t.Errorf("werkwoordopgave zonder persoon of met de 3e persoon (staat al op de kaart): %+v", ex)
		}
	}
	if !seen["ban:ház"] || !seen["ww-onbep:olvas"] || len(seen) != 2 {
		t.Errorf("opgaven: %v", seen)
	}
	if _, ok := Next(words, []string{"ww-bep"}[:0], nil, "", rng); ok {
		t.Error("zonder actieve regels geen opgave")
	}
}

func TestCheckAndScores(t *testing.T) {
	if r, _ := Check("ban", "ház", 0, " Házban "); !r.Correct {
		t.Errorf("házban moet goed zijn: %+v", r)
	}
	if r, _ := Check("ban", "ház", 0, "hazban"); r.Correct || !r.AccentOnly || r.Form.Text != "házban" {
		t.Errorf("hazban: alleen accenten fout: %+v", r)
	}
	if r, _ := Check("ww-onbep", "olvas", int(Sg2), "olvassz"); r.Correct || r.Form.Text != "olvasol" || len(r.Form.Steps) == 0 {
		t.Errorf("olvas, te: %+v", r)
	}
	if _, ok := Check("bestaat-niet", "ház", 0, "x"); ok {
		t.Error("onbekende regel")
	}
	var log []storage.GrammarAttempt
	for i := range 25 {
		log = append(log, storage.GrammarAttempt{Rule: "ban", Correct: i >= 10})
	}
	s := Scores(log)["ban"]
	if s.Attempts != 25 || s.Correct != 15 || s.RecentN != 20 || s.RecentCorrect != 15 {
		t.Errorf("score: %+v", s)
	}
}

func TestExamplesVary(t *testing.T) {
	words := []Word{{Hungarian: "ház", Type: "zn", Known: true}, {Hungarian: "bolt", Type: "zn", Known: true}, {Hungarian: "kert", Type: "zn"}}
	r, _ := RuleByID("ban")
	ex := r.Examples(words, 2)
	if len(ex) != 2 || ex[0].Form != "házban" || ex[1].Form != "kertben" {
		t.Errorf("voorbeelden moeten -ban én -ben laten zien: %+v", ex)
	}
}
