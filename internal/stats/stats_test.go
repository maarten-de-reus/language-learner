package stats

import (
	"fmt"
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

var ams = mustLoc("Europe/Amsterdam")

func mustLoc(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}

// at geeft een moment in oktober 2026, Amsterdamse tijd
func at(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, ams) }

func testLib() *storage.Library {
	return &storage.Library{
		Words: map[string]storage.Word{
			"w_nl_koffie": {ID: "w_nl_koffie", Language: "nl", Text: "koffie"},
			"w_hu_kave":   {ID: "w_hu_kave", Language: "hu", Text: "kávé"},
			"w_nl_water":  {ID: "w_nl_water", Language: "nl", Text: "water"},
			"w_hu_viz":    {ID: "w_hu_viz", Language: "hu", Text: "víz"},
		},
		Concepts: []storage.TranslationConcept{
			{ID: "c_koffie", SourceWordID: "w_nl_koffie", TargetWordIDs: []string{"w_hu_kave"}, StepIndex: 1, NextReview: at(20, 4), FirstReview: at(1, 9)}, // te laat
			{ID: "c_water", SourceWordID: "w_nl_water", ContextSource: "drinken", TargetWordIDs: []string{"w_hu_viz"}, NextReview: at(26, 4), FirstReview: at(1, 9)},
		},
	}
}

func review(t time.Time, id string, correct bool, opts ...func(*storage.Review)) storage.Review {
	r := storage.Review{Time: t, ConceptID: id, Correct: correct}
	for _, o := range opts {
		o(&r)
	}
	return r
}

func retry(r *storage.Review)  { r.Retry = true }
func accent(r *storage.Review) { r.AccentOnly = true }

func TestTotalsAndMistakes(t *testing.T) {
	reviews := []storage.Review{
		review(at(23, 20), "c_koffie", false, accent),
		review(at(23, 20), "c_koffie", true, retry), // telt niet mee voor de score
		review(at(23, 21), "c_water", true),
		review(at(24, 9), "c_water", false),
	}
	rep := Compute(testLib(), reviews, at(24, 12), 20)

	if rep.Answers != 4 || rep.FirstAttempts != 3 || rep.FirstCorrect != 1 || rep.WordsPracticed != 2 {
		t.Errorf("totalen: %+v", rep)
	}
	if rep.Accuracy < 0.33 || rep.Accuracy > 0.34 {
		t.Errorf("score: %v, verwacht 1/3", rep.Accuracy)
	}
	if rep.Mistakes != (Mistakes{AccentOnly: 1, Other: 1}) {
		t.Errorf("fouten: %+v", rep.Mistakes)
	}
	if !rep.PracticedToday || rep.Today.Wrong != 1 || rep.Today.Correct != 0 {
		t.Errorf("vandaag: %+v", rep.Today)
	}
}

func TestStreaks(t *testing.T) {
	days := func(ds ...int) []storage.Review {
		var rs []storage.Review
		for _, d := range ds {
			rs = append(rs, review(at(d, 20), "c_water", true))
		}
		return rs
	}
	cases := []struct {
		name             string
		reviews          []storage.Review
		now              time.Time
		current, longest int
	}{
		{"geen antwoorden", nil, at(24, 12), 0, 0},
		{"drie dagen tot vandaag", days(22, 23, 24), at(24, 21), 3, 3},
		{"vandaag nog niet geoefend: reeks loopt nog", days(22, 23), at(24, 12), 2, 2},
		{"eergisteren voor het laatst: reeks voorbij", days(20, 21, 22), at(24, 12), 0, 3},
		{"onderbroken", days(15, 16, 17, 18, 22, 23), at(23, 21), 2, 4},
		// 01:00 op de 24e hoort nog bij de 23e
		{"na middernacht telt als de vorige dag", append(days(22), review(at(24, 1), "c_water", true)), at(24, 2), 2, 2},
	}
	for _, c := range cases {
		rep := Compute(testLib(), c.reviews, c.now, 20)
		if rep.Streak != c.current || rep.LongestStreak != c.longest {
			t.Errorf("%s: reeks %d / langste %d, verwacht %d / %d", c.name, rep.Streak, rep.LongestStreak, c.current, c.longest)
		}
	}
}

func TestActivityAndForecast(t *testing.T) {
	reviews := []storage.Review{review(at(24, 9), "c_water", true), review(at(20, 9), "c_koffie", false)}
	rep := Compute(testLib(), reviews, at(24, 12), 20)

	if len(rep.Activity) != ActivityDays || rep.Activity[ActivityDays-1].Date != "2026-10-24" {
		t.Fatalf("activiteit moet %d dagen tot vandaag bevatten, kreeg %d (laatste %v)", ActivityDays, len(rep.Activity), rep.Activity[len(rep.Activity)-1])
	}
	if a := rep.Activity[ActivityDays-5]; a.Date != "2026-10-20" || a.Wrong != 1 {
		t.Errorf("20 oktober: %+v", a)
	}

	if len(rep.Forecast) != ForecastDays || rep.Forecast[0].Date != "2026-10-24" {
		t.Fatalf("vooruitblik: %+v", rep.Forecast)
	}
	if rep.Forecast[0].Due != 1 || rep.Forecast[2].Due != 1 {
		t.Errorf("verwacht koffie vandaag (achterstand) en water over 2 dagen, kreeg %+v", rep.Forecast[:3])
	}
}

func TestHardest(t *testing.T) {
	reviews := []storage.Review{
		review(at(20, 9), "c_water", false),
		review(at(21, 9), "c_water", false),
		review(at(22, 9), "c_water", true),
		review(at(22, 9), "c_koffie", false),
		review(at(22, 9), "c_koffie", true, retry), // herhalingen tellen niet
		review(at(22, 9), "c_verwijderd", false),   // bestaat niet meer
	}
	rep := Compute(testLib(), reviews, at(24, 12), 20)
	if len(rep.Hardest) != 2 {
		t.Fatalf("verwacht 2 moeilijke woorden, kreeg %+v", rep.Hardest)
	}
	w := rep.Hardest[0]
	if w.ID != "c_water" || w.Wrong != 2 || w.Attempts != 3 || w.Prompt != "water" || w.Context != "drinken" || w.Answers[0] != "víz" {
		t.Errorf("moeilijkste: %+v", w)
	}
	if rep.Hardest[1].ID != "c_koffie" || rep.Hardest[1].Attempts != 1 {
		t.Errorf("tweede: %+v", rep.Hardest[1])
	}
}

func TestRetryEffect(t *testing.T) {
	reviews := []storage.Review{
		// Fout, goed bij herhaling, volgende dag goed
		review(at(20, 9), "c_water", false), review(at(20, 9), "c_water", true, retry), review(at(21, 9), "c_water", true),
		// Fout, goed bij herhaling, volgende dag weer fout
		review(at(20, 9), "c_koffie", false), review(at(20, 9), "c_koffie", true, retry), review(at(22, 9), "c_koffie", false),
		// Fout en ook bij herhaling fout: telt niet
		review(at(23, 9), "c_koffie", false, accent), review(at(23, 9), "c_koffie", false, retry),
		// Fout, goed bij herhaling, nog niet opnieuw getoetst: telt (nog) niet
		review(at(24, 9), "c_water", false), review(at(24, 9), "c_water", true, retry),
	}
	rep := Compute(testLib(), reviews, at(24, 12), 20)
	if rep.Retry.Cases != 2 || rep.Retry.NextCorrect != 1 || rep.Retry.Enough {
		t.Errorf("herhaaleffect: %+v", rep.Retry)
	}
}

func TestPerConcept(t *testing.T) {
	counts := PerConcept([]storage.Review{
		review(at(20, 9), "c_water", false),
		review(at(20, 9), "c_water", true, retry),
		review(at(21, 9), "c_water", true),
	})
	if counts["c_water"] != (Counts{Attempts: 2, Wrong: 1}) {
		t.Errorf("tellingen: %+v", counts)
	}
}

func TestQueue(t *testing.T) {
	lib := testLib()
	for i := range 45 {
		lib.Concepts = append(lib.Concepts, storage.TranslationConcept{ID: fmt.Sprintf("c_new_%d", i), SourceWordID: "w_nl_water", TargetWordIDs: []string{"w_hu_viz"}, NextReview: at(24, 9)})
	}
	rep := Compute(lib, nil, at(24, 12), 20)
	if rep.Queue != (Queue{Waiting: 45, PerDay: 20, Days: 3}) {
		t.Errorf("wachtrij: %+v", rep.Queue)
	}
	if rep.Forecast[0].Due != 1 {
		t.Errorf("nieuwe woorden horen niet in de planning; vandaag alleen koffie, kreeg %d", rep.Forecast[0].Due)
	}
	if q := Compute(lib, nil, at(24, 12), 0).Queue; q.Days != 0 || q.Waiting != 45 {
		t.Errorf("zonder limiet: %+v", q)
	}
}

func TestDirectionsSplitNewFromReview(t *testing.T) {
	lib := testLib()
	lib.Concepts = append(lib.Concepts, storage.TranslationConcept{ID: "c_kave", SourceWordID: "w_hu_kave", TargetWordIDs: []string{"w_nl_koffie"}})
	reviews := []storage.Review{
		review(at(20, 9), "c_koffie", false), // nl -> hu, eerste keer
		review(at(20, 9), "c_koffie", true, retry),
		review(at(21, 9), "c_koffie", true), // nl -> hu, herhaling
		review(at(22, 9), "c_koffie", false),
		review(at(20, 9), "c_kave", true), // hu -> nl, eerste keer
		review(at(23, 9), "c_kave", true),
		review(at(23, 9), "c_weg", true), // verwijderd concept telt niet mee
	}
	got := Compute(lib, reviews, at(24, 12), 20).Directions
	want := []Direction{
		{From: "hu", To: "nl", New: Score{1, 1}, Review: Score{1, 1}},
		{From: "nl", To: "hu", New: Score{1, 0}, Review: Score{2, 1}},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("richtingen:\n kreeg   %+v\n verwacht %+v", got, want)
	}
}

func TestLapsesFollowTheNextTwoReviews(t *testing.T) {
	step := func(before, after int) func(*storage.Review) {
		return func(r *storage.Review) { r.StepBefore, r.StepAfter = before, after }
	}
	reviews := []storage.Review{
		// a: vergeten op 40 dagen (4 -> 2), morgen goed, daarna op 16 dagen weer goed
		review(at(1, 9), "a", false, step(4, 2)),
		review(at(1, 9), "a", true, retry), // oefenherhaling telt niet als volgende toets
		review(at(2, 9), "a", true, step(2, 3)),
		review(at(18, 9), "a", true, step(3, 4)),
		// b: vergeten op 7 dagen (2 -> 0), morgen weer fout
		review(at(3, 9), "b", false, step(2, 0)),
		review(at(4, 9), "b", false, step(0, 0)),
		// c: vergeten, nog niet opnieuw gevraagd
		review(at(20, 9), "c", false, step(5, 3)),
		// telt niet mee: volledige reset van vóór de regel, accentfout, nog aan het leren
		review(at(5, 9), "d", false, step(4, 0)),
		review(at(5, 9), "e", false, step(4, 3), accent),
		review(at(5, 9), "f", false, step(1, 0)),
	}
	got := Compute(testLib(), reviews, at(24, 12), 20).Lapses
	want := Lapses{Cases: 3, Retested: 2, NextCorrect: 1, Kept: 1, KeptCorrect: 1}
	if got != want {
		t.Errorf("vergeten woorden:\n kreeg    %+v\n verwacht %+v", got, want)
	}
}
