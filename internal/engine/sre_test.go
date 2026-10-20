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
