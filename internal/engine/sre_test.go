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
		chunk, remaining := GetNextChunk(pairs(15, 0), 10)
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
	chunk, _ := GetNextChunk(pairs(15, 0), 10)
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
		chunk, _ := GetNextChunk(cs, 5)
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

func TestDeferredDirectionComesInLaterRun(t *testing.T) {
	cs := pairs(1, 0) // alleen koffie -> kávé en kávé -> koffie
	chunk, remaining := GetNextChunk(cs, 10)
	if len(chunk) != 1 || remaining != 1 {
		t.Fatalf("verwacht 1 in de run en 1 uitgesteld, kreeg %d / %d", len(chunk), remaining)
	}

	// Na het oefenen is de eerste niet meer due; de andere richting komt nu aan de beurt
	for i := range cs {
		if cs[i].ID == chunk[0].ID {
			cs[i].NextReview = time.Now().Add(24 * time.Hour)
		}
	}
	next, remaining := GetNextChunk(cs, 10)
	if len(next) != 1 || next[0].ID == chunk[0].ID || remaining != 0 {
		t.Errorf("verwacht de andere richting, kreeg %v (over: %d)", ids(next), remaining)
	}
}

func TestNotDueIsExcluded(t *testing.T) {
	cs := pairs(3, 0)
	cs[0].NextReview = time.Now().Add(time.Hour)
	chunk, _ := GetNextChunk(cs, 10)
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
	updateProgress(&wrong, false, now)
	if wrong.StepIndex != 0 || !wrong.NextReview.Equal(tomorrow) {
		t.Errorf("fout: verwacht stap 0 en morgen %02d:00, kreeg %+v", DayStartHour, wrong)
	}

	right := storage.TranslationConcept{StepIndex: 0, NextReview: now.Add(-time.Hour)}
	updateProgress(&right, true, now)
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
		updateProgress(&c, true, time.Now())
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
	updateProgress(&early, true, now)
	if early.StepIndex != before.StepIndex || !early.NextReview.Equal(before.NextReview) {
		t.Errorf("te vroeg goed mag niets veranderen: %+v -> %+v", before, early)
	}

	// Te vroeg en fout: toch terug naar stap 0
	updateProgress(&early, false, now)
	if early.StepIndex != 0 {
		t.Errorf("te vroeg fout moet terug naar stap 0, kreeg %d", early.StepIndex)
	}

	// Op de hoogste stap blijft het bij de hoogste stap
	top := storage.TranslationConcept{StepIndex: len(AnkiIntervals) - 1, NextReview: past}
	updateProgress(&top, true, now)
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
