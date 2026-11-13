package storage

import (
	"slices"
	"testing"
)

func TestReverses(t *testing.T) {
	concepts := []TranslationConcept{
		{ID: "hallo", SourceWordID: "nl_hallo", TargetWordIDs: []string{"hu_szia"}},
		{ID: "hoi", SourceWordID: "nl_hoi", TargetWordIDs: []string{"hu_szia"}},
		{ID: "szia", SourceWordID: "hu_szia", TargetWordIDs: []string{"nl_hallo", "nl_hoi"}},
		{ID: "half", SourceWordID: "hu_szia", TargetWordIDs: []string{"nl_dag"}}, // raakt hallo alleen van één kant
	}
	r := NewReverses(concepts)

	if got := r.Of(concepts[2]); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("szia -> hallo, hoi heeft twee omgekeerde richtingen, kreeg %v", got)
	}
	if got := r.Of(concepts[0]); !slices.Equal(got, []int{2}) {
		t.Errorf("hallo -> szia: alleen szia telt strikt als omgekeerde richting, kreeg %v", got)
	}
	if got := r.Related(concepts[0]); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("ruim: hallo raakt szia en de halve kaart, kreeg %v", got)
	}
	if !Related(concepts[0], concepts[3]) || Related(concepts[0], concepts[1]) {
		t.Error("Related voor twee kaarten: hallo raakt de halve kaart, hallo en hoi raken elkaar niet")
	}
}
