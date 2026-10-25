package catalog

import (
	"slices"
	"strings"
	"testing"
)

func TestNormalizeGroups(t *testing.T) {
	lib := newLib()
	add(t, lib, "koffie", "", "kávé", func(nc *NewPair) { nc.Groups = []string{"Eten en drinken"} })

	got, err := NormalizeGroups(lib, []string{"  eten   EN drinken ", "Week 1", "week 1", ""})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Eten en drinken", "Week 1"}; !slices.Equal(got, want) {
		t.Errorf("verwacht bestaande schrijfwijze hergebruikt en dubbelen weg: %v, kreeg %v", want, got)
	}
	if _, err := NormalizeGroups(lib, []string{strings.Repeat("x", MaxGroupName+1)}); err == nil {
		t.Error("te lange groepsnaam moet geweigerd worden")
	}
	if _, err := NormalizeGroups(lib, []string{"a, b"}); err == nil {
		t.Error("een komma in een groepsnaam moet geweigerd worden")
	}
}

func TestGroupsOnBothDirectionsAndCounts(t *testing.T) {
	lib := newLib()
	add(t, lib, "koffie", "", "kávé", func(nc *NewPair) { nc.Groups = []string{"Drinken", "Week 1"} })
	add(t, lib, "thee", "", "tea", func(nc *NewPair) { nc.Groups = []string{"drinken"} })

	for _, c := range lib.Concepts[:2] {
		if !slices.Equal(c.Groups, []string{"Drinken", "Week 1"}) {
			t.Errorf("%s: beide richtingen moeten de groepen krijgen, kreeg %v", c.ID, c.Groups)
		}
	}
	if got := Groups(lib); !slices.Equal(got, []Group{{"Drinken", 4}, {"Week 1", 2}}) {
		t.Errorf("groepen met aantallen: %+v", got)
	}
}

func TestEditGroupsFollowsReverse(t *testing.T) {
	lib := newLib()
	add(t, lib, "koffie", "", "kávé", func(nc *NewPair) { nc.Groups = []string{"Drinken", "Week 1"} })
	nlhu, hunl := lib.Concepts[0].ID, lib.Concepts[1].ID

	// Week 1 eraf, Ochtend erbij: geldt ook voor de andere richting
	if err := Update(lib, nlhu, Edit{Answers: []string{"kávé"}, Groups: []string{"Drinken", "Ochtend"}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{nlhu, hunl} {
		if got := concept(t, lib, id).Groups; !slices.Equal(got, []string{"Drinken", "Ochtend"}) {
			t.Errorf("%s: kreeg %v", id, got)
		}
	}
	// Zonder Groups in de bewerking blijven de groepen staan
	if err := Update(lib, nlhu, Edit{Answers: []string{"kávé"}, Example: "Kérek egy kávét."}); err != nil {
		t.Fatal(err)
	}
	if got := concept(t, lib, nlhu).Groups; len(got) != 2 {
		t.Errorf("groepen moeten blijven staan als ze niet meegestuurd worden, kreeg %v", got)
	}
	// Leeg = alle groepen eraf
	if err := Update(lib, nlhu, Edit{Answers: []string{"kávé"}, Groups: []string{}}); err != nil {
		t.Fatal(err)
	}
	if g := concept(t, lib, hunl).Groups; g != nil {
		t.Errorf("alle groepen weg, ook in de andere richting, kreeg %v", g)
	}
}

func TestImportAppliesGroups(t *testing.T) {
	lib := newLib()
	results, ok := Import(lib, ParseImport("kávé;koffie\ntea;thee"), nil, []string{"Week 2"})
	if !ok || len(results) != 2 {
		t.Fatalf("import: %+v", results)
	}
	for _, c := range lib.Concepts {
		if !slices.Equal(c.Groups, []string{"Week 2"}) {
			t.Errorf("%s: verwacht groep Week 2, kreeg %v", c.ID, c.Groups)
		}
	}
}
