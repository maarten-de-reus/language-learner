package engine

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

var past = time.Now().Add(-time.Hour)

// pairs maakt n woordparen, elk in beide richtingen (zoals de migratie: NL->HU direct gevolgd door HU->NL)
func pairs(n, step int) []storage.TranslationConcept {
	var cs []storage.TranslationConcept
	for i := 0; i < n; i++ {
		nl, hu := fmt.Sprintf("w_nl_%d", i), fmt.Sprintf("w_hu_%d", i)
		cs = append(cs,
			storage.TranslationConcept{ID: fmt.Sprintf("c_nl_hu_%d", i), SourceWordID: nl, TargetWordIDs: []string{hu}, StepIndex: step, NextReview: past},
			storage.TranslationConcept{ID: fmt.Sprintf("c_hu_nl_%d", i), SourceWordID: hu, TargetWordIDs: []string{nl}, StepIndex: step, NextReview: past},
		)
	}
	return cs
}

func ids(cs []storage.TranslationConcept) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}

func TestChunkNeverContainsBothDirections(t *testing.T) {
	for range 50 {
		chunk, remaining := GetNextChunk(seen(pairs(15, 0)), 10, Unlimited)
		if len(chunk) != 10 || remaining != 20 {
			t.Fatalf("verwacht 10 in de run en 20 over, kreeg %d / %d", len(chunk), remaining)
		}
		for i, a := range chunk {
			for _, b := range chunk[i+1:] {
				if reverseOf(a, b) {
					t.Fatalf("beide richtingen in één run: %s en %s", a.ID, b.ID)
				}
			}
		}
	}
}

func TestChunkIsRandomised(t *testing.T) {
	first := strings.Join(ids(mustChunk(t)), ",")
	for range 20 {
		if strings.Join(ids(mustChunk(t)), ",") != first {
			return
		}
	}
	t.Error("20 keer dezelfde run in dezelfde volgorde: de selectie wordt niet gehusseld")
}

func mustChunk(t *testing.T) []storage.TranslationConcept {
	t.Helper()
	chunk, _ := GetNextChunk(pairs(15, 0), 10, Unlimited)
	return chunk
}

func TestLowerStepsComeFirst(t *testing.T) {
	cs := append(pairs(10, 2), pairs(0, 0)...)
	for i := range 3 {
		cs = append(cs, storage.TranslationConcept{
			ID: fmt.Sprintf("c_urgent_%d", i), SourceWordID: fmt.Sprintf("w_u_%d", i), TargetWordIDs: []string{fmt.Sprintf("w_v_%d", i)}, NextReview: past,
		})
	}
	for range 20 {
		chunk, _ := GetNextChunk(cs, 5, Unlimited)
		urgent := 0
		for _, c := range chunk {
			if strings.HasPrefix(c.ID, "c_urgent_") {
				urgent++
			}
		}
		if urgent != 3 {
			t.Fatalf("alle 3 concepten op stap 0 moeten in de run, kreeg %d: %v", urgent, ids(chunk))
		}
	}
}

func TestOtherDirectionWaitsUntilTomorrow(t *testing.T) {
	cs := seen(pairs(1, 0)) // alleen koffie -> kávé en kávé -> koffie, allebei bekend en aan de beurt
	chunk, remaining := GetNextChunk(cs, 10, Unlimited)
	if len(chunk) != 1 || remaining != 1 {
		t.Fatalf("verwacht 1 in de run en 1 uitgesteld, kreeg %d / %d", len(chunk), remaining)
	}

	// De eerste richting is vandaag beantwoord (goed of fout): de andere wacht tot morgen
	for i := range cs {
		if cs[i].ID == chunk[0].ID {
			cs[i].NextReview = time.Now().Add(24 * time.Hour)
			cs[i].LastReview = time.Now()
		}
	}
	if next, remaining := GetNextChunk(cs, 10, Unlimited); len(next) != 0 || remaining != 0 {
		t.Errorf("de andere richting mag vandaag niet meer komen, kreeg %v (over: %d)", ids(next), remaining)
	}

	// Gisteren beantwoord: dan mag de andere richting vandaag wel
	for i := range cs {
		if cs[i].ID == chunk[0].ID {
			cs[i].LastReview = time.Now().AddDate(0, 0, -1)
		}
	}
	if next, _ := GetNextChunk(cs, 10, Unlimited); len(next) != 1 || next[0].ID == chunk[0].ID {
		t.Errorf("verwacht de andere richting, kreeg %v", ids(next))
	}
}

func TestNotDueIsExcluded(t *testing.T) {
	cs := pairs(3, 0)
	cs[0].NextReview = time.Now().Add(time.Hour)
	chunk, _ := GetNextChunk(cs, 10, Unlimited)
	if slices.Contains(ids(chunk), cs[0].ID) {
		t.Error("concept dat nog niet due is mag niet in de run")
	}
}

func TestReviewDateUsesDayBoundary(t *testing.T) {
	ams, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Skip("tijdzonedata niet beschikbaar")
	}
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, ams) }

	cases := []struct {
		name string
		now  time.Time
		days int
		want time.Time
	}{
		{"avond oefenen, morgen herhalen: morgenochtend", at(20, 21, 0), 1, at(21, 4, 0)},
		{"ochtend oefenen, morgen herhalen", at(20, 9, 30), 1, at(21, 4, 0)},
		{"na middernacht hoort nog bij de vorige dag", at(21, 1, 30), 1, at(21, 4, 0)},
		{"precies om 04:00 begint de nieuwe dag", at(21, 4, 0), 1, at(22, 4, 0)},
		{"7 dagen, over de wintertijd heen", at(20, 21, 0), 7, at(27, 4, 0)},
	}
	for _, c := range cases {
		if got := ReviewDate(c.now, c.days); !got.Equal(c.want) {
			t.Errorf("%s: kreeg %v, verwacht %v", c.name, got, c.want)
		}
	}
}

func TestUpdateProgressSchedulesAtStartOfDay(t *testing.T) {
	now := time.Date(2026, 10, 20, 21, 15, 0, 0, time.Local)
	tomorrow := time.Date(2026, 10, 21, DayStartHour, 0, 0, 0, time.Local)

	wrong := storage.TranslationConcept{StepIndex: 3, NextReview: now.Add(-time.Hour)}
	updateProgress(&wrong, false, now, nil)
	if wrong.StepIndex != 0 || !wrong.NextReview.Equal(tomorrow) {
		t.Errorf("fout: verwacht stap 0 en morgen %02d:00, kreeg %+v", DayStartHour, wrong)
	}

	right := storage.TranslationConcept{StepIndex: 0, NextReview: now.Add(-time.Hour)}
	updateProgress(&right, true, now, nil)
	want := time.Date(2026, 10, 23, DayStartHour, 0, 0, 0, time.Local) // stap 1 = 3 dagen
	if right.StepIndex != 1 || !right.NextReview.Equal(want) {
		t.Errorf("goed: verwacht stap 1 en %v, kreeg %+v", want, right)
	}
}

func TestIntervalFuzzStaysWithinBounds(t *testing.T) {
	bounds := map[int][2]int{0: {1, 1}, 1: {3, 3}, 2: {6, 8}, 3: {14, 18}, 4: {38, 42}, 5: {86, 95}}
	for step, b := range bounds {
		for range 500 {
			if got := CalculateIntervalWithFuzz(step); got < b[0] || got > b[1] {
				t.Fatalf("stap %d: interval %d buiten [%d, %d]", step, got, b[0], b[1])
			}
		}
	}
}

func TestCorruptStepDoesNotCrash(t *testing.T) {
	for _, step := range []int{-3, 6, 99} {
		_ = CalculateIntervalWithFuzz(step) // mag niet panieken

		c := storage.TranslationConcept{StepIndex: step, NextReview: past}
		updateProgress(&c, true, time.Now(), nil)
		if c.StepIndex < 0 || c.StepIndex >= len(AnkiIntervals) {
			t.Errorf("stap %d: na een antwoord moet de stap geldig zijn, kreeg %d", step, c.StepIndex)
		}
	}
}

func TestUpdateProgressEdgeCases(t *testing.T) {
	now := time.Now()

	// Te vroeg en goed: niets verandert
	early := storage.TranslationConcept{StepIndex: 2, NextReview: now.Add(48 * time.Hour)}
	before := early
	updateProgress(&early, true, now, nil)
	if early.StepIndex != before.StepIndex || !early.NextReview.Equal(before.NextReview) {
		t.Errorf("te vroeg goed mag niets veranderen: %+v -> %+v", before, early)
	}

	// Te vroeg en fout: toch terug naar stap 0
	updateProgress(&early, false, now, nil)
	if early.StepIndex != 0 {
		t.Errorf("te vroeg fout moet terug naar stap 0, kreeg %d", early.StepIndex)
	}

	// Op de hoogste stap blijft het bij de hoogste stap
	top := storage.TranslationConcept{StepIndex: len(AnkiIntervals) - 1, NextReview: past}
	updateProgress(&top, true, now, nil)
	if top.StepIndex != len(AnkiIntervals)-1 {
		t.Errorf("hoogste stap moet blijven, kreeg %d", top.StepIndex)
	}
}

func TestDayOfAndDaysBetween(t *testing.T) {
	ams, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Skip("tijdzonedata niet beschikbaar")
	}
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, ams) }
	if got := DayOf(at(24, 1)); !got.Equal(time.Date(2026, 10, 23, 0, 0, 0, 0, ams)) {
		t.Errorf("01:00 hoort bij de vorige dag, kreeg %v", got)
	}
	if got := DayOf(at(24, 4)); !got.Equal(time.Date(2026, 10, 24, 0, 0, 0, 0, ams)) {
		t.Errorf("04:00 is de nieuwe dag, kreeg %v", got)
	}
	// Over de overgang naar wintertijd (25 oktober) heen
	if got := DaysBetween(DayOf(at(24, 12)), DayOf(at(27, 12))); got != 3 {
		t.Errorf("verwacht 3 dagen, kreeg %d", got)
	}
}

// seen markeert concepten als al eens gevraagd (geen nieuwe woorden meer)
func seen(cs []storage.TranslationConcept) []storage.TranslationConcept {
	for i := range cs {
		cs[i].FirstReview = past
	}
	return cs
}

func newWords(n int) []storage.TranslationConcept {
	var cs []storage.TranslationConcept
	for i := range n {
		cs = append(cs, storage.TranslationConcept{
			ID: fmt.Sprintf("c_new_%02d", i), SourceWordID: fmt.Sprintf("w_a_%d", i), TargetWordIDs: []string{fmt.Sprintf("w_b_%d", i)}, NextReview: past,
		})
	}
	return cs
}

func TestNewWordsAreLimited(t *testing.T) {
	cs := append(seen(pairs(3, 1)), newWords(20)...) // 6 bekende herhalingen + 20 nieuwe woorden

	chunk, remaining := GetNextChunk(cs, 50, 5)
	var news []string
	for _, c := range chunk {
		if IsNew(c) {
			news = append(news, c.ID)
		}
	}
	slices.Sort(news)
	if want := []string{"c_new_00", "c_new_01", "c_new_02", "c_new_03", "c_new_04"}; !slices.Equal(news, want) {
		t.Errorf("verwacht de eerste 5 nieuwe woorden (volgorde van toevoegen), kreeg %v", news)
	}
	if len(chunk) != 3+5 || remaining != 3 { // 3 van de 6 herhalingen zijn de andere richting
		t.Errorf("verwacht 8 in de run en 3 uitgesteld, kreeg %d / %d", len(chunk), remaining)
	}

	if chunk, _ := GetNextChunk(cs, 50, 0); slices.ContainsFunc(chunk, IsNew) {
		t.Error("met 0 toegestaan mogen er geen nieuwe woorden in de run")
	}
	if chunk, _ := GetNextChunk(cs, 50, Unlimited); len(chunk) != 3+20 {
		t.Errorf("zonder limiet verwacht 23 in de run, kreeg %d", len(chunk))
	}
}

func TestIntervalRangeMatchesFuzz(t *testing.T) {
	want := map[int][2]int{0: {1, 1}, 1: {3, 3}, 2: {6, 8}, 3: {14, 18}, 4: {38, 42}, 5: {86, 95}}
	for step, w := range want {
		if lo, hi := IntervalRange(step); lo != w[0] || hi != w[1] {
			t.Errorf("stap %d: [%d, %d], verwacht %v", step, lo, hi, w)
		}
	}
}

func TestLoadBalancingPicksQuietestDay(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.Local)
	day := func(d int) time.Time { return DayOf(ReviewDate(now, d)) }

	// Stap 2 (7 dagen, 6 tot 8): dag 7 is druk, dag 6 en 8 zijn even rustig
	busy := map[time.Time]int{day(6): 3, day(7): 40, day(8): 3}
	load := func(d time.Time) int { return busy[d] }
	counts := map[int]int{}
	for range 200 {
		counts[balancedInterval(2, now, load)]++
	}
	if counts[7] != 0 || counts[6] == 0 || counts[8] == 0 {
		t.Errorf("verwacht nooit de drukke dag 7 en willekeurig 6 of 8, kreeg %v", counts)
	}

	// Gelijke drukte: het basis-interval wint
	if got := balancedInterval(2, now, func(time.Time) int { return 5 }); got != 7 {
		t.Errorf("bij gelijke drukte verwacht 7 dagen, kreeg %d", got)
	}
	// Korte intervallen liggen vast, ook als die dag druk is
	if got := balancedInterval(1, now, load); got != 3 {
		t.Errorf("3 dagen heeft geen speling, kreeg %d", got)
	}
	// Zonder load: willekeurig binnen de grenzen, zoals voorheen
	for range 100 {
		if got := balancedInterval(3, now, nil); got < 14 || got > 18 {
			t.Fatalf("zonder load buiten [14, 18]: %d", got)
		}
	}
}

func TestNewWordsOneDirectionPerDay(t *testing.T) {
	// 15 nieuwe woorden in beide richtingen, zoals na een import (NL->HU direct gevolgd door HU->NL)
	cs := pairs(15, 0)
	cands := DueCandidates(cs, time.Now(), 10)
	if len(cands) != 10 {
		t.Fatalf("verwacht 10 nieuwe kaarten, kreeg %d", len(cands))
	}
	for i, a := range cands {
		for _, b := range cands[i+1:] {
			if reverseOf(a, b) {
				t.Fatalf("beide richtingen van een nieuw woord op één dag: %s en %s", a.ID, b.ID)
			}
		}
	}
	// "Nog 10" geeft dus 10 verschillende woorden, en die passen allemaal in één ronde
	if chunk, _ := GetNextChunk(cs, 10, 10); len(chunk) != 10 {
		t.Errorf("verwacht een ronde met 10 nieuwe woorden, kreeg %d", len(chunk))
	}
}

func TestBuried(t *testing.T) {
	now := time.Now()
	cs := seen(pairs(2, 1))
	cs[0].LastReview = now // koffie_0 NL->HU vandaag beantwoord
	b := Buried(cs, now)
	if !b["c_hu_nl_0"] || b["c_nl_hu_0"] || b["c_nl_hu_1"] || b["c_hu_nl_1"] {
		t.Errorf("alleen de andere richting van woord 0 hoort te wachten, kreeg %v", b)
	}
	cs[0].LastReview = now.AddDate(0, 0, -1)
	if b := Buried(cs, now); len(b) != 0 {
		t.Errorf("een antwoord van gisteren laat niets wachten, kreeg %v", b)
	}
}
