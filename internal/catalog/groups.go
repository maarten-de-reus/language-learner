package catalog

import (
	"fmt"
	"slices"
	"strings"

	"hungarian-language-learner/internal/storage"
)

// MaxGroupName is de maximale lengte van een groepsnaam
const MaxGroupName = 40

// Group is een groep met het aantal concepten erin
type Group struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Groups geeft alle groepen die in gebruik zijn, alfabetisch
func Groups(lib *storage.Library) []Group {
	counts := map[string]int{}
	for _, c := range lib.Concepts {
		for _, g := range c.Groups {
			counts[g]++
		}
	}
	out := make([]Group, 0, len(counts))
	for name, n := range counts {
		out = append(out, Group{Name: name, Count: n})
	}
	slices.SortFunc(out, func(a, b Group) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
}

// NormalizeGroups schoont groepsnamen op: spaties aan de randen weg, dubbele spaties samen,
// dubbelen eruit, en een bestaande groep wordt hergebruikt als alleen de hoofdletters verschillen
// ("eten" wordt "Eten" als die groep al bestaat). Een nieuwe naam maakt de groep aan.
func NormalizeGroups(lib *storage.Library, names []string) ([]string, error) {
	existing := Groups(lib)
	var out []string
	for _, n := range names {
		n = strings.Join(strings.Fields(n), " ")
		if n == "" {
			continue
		}
		if len([]rune(n)) > MaxGroupName {
			return nil, fmt.Errorf("groepsnaam %q is te lang (maximaal %d tekens)", n, MaxGroupName)
		}
		if strings.Contains(n, ",") {
			return nil, fmt.Errorf("groepsnaam %q mag geen komma bevatten", n)
		}
		for _, g := range existing {
			if strings.EqualFold(g.Name, n) {
				n = g.Name
				break
			}
		}
		if !slices.ContainsFunc(out, func(x string) bool { return strings.EqualFold(x, n) }) {
			out = append(out, n)
		}
	}
	return out, nil
}

// addGroups voegt groepen toe aan een concept (bestaande blijven staan)
func addGroups(c *storage.TranslationConcept, groups []string) {
	for _, g := range groups {
		if !slices.Contains(c.Groups, g) {
			c.Groups = append(c.Groups, g)
		}
	}
}

// removeGroups haalt groepen weg uit een concept
func removeGroups(c *storage.TranslationConcept, groups []string) {
	c.Groups = slices.DeleteFunc(c.Groups, func(g string) bool { return slices.Contains(groups, g) })
	if len(c.Groups) == 0 {
		c.Groups = nil
	}
}
