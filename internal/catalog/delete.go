// Kaarten verwijderen, met of zonder de omgekeerde richting.

package catalog

import (
	"slices"

	"hungarian-language-learner/internal/storage"
)

// DeleteResult beschrijft wat Delete heeft gedaan
type DeleteResult struct {
	Deleted []string `json:"deleted"` // verwijderde concepten
	Updated []string `json:"updated"` // omgekeerde concepten waar alleen een antwoord uit is gehaald
}

// Delete verwijdert een concept, en met withReverse ook de omgekeerde richting.
// Een omgekeerd concept dat ook bij andere woorden hoort (szia -> hallo, hoi) wordt niet verwijderd;
// alleen dit woord gaat eruit als goed antwoord. Woorden die nergens meer gebruikt worden verdwijnen.
// Verwijderde concepten moeten daarna ook uit de lopende quiz (quiz.Session.Forget).
func Delete(lib *storage.Library, id string, withReverse bool) (DeleteResult, error) {
	var res DeleteResult
	i := lib.Index(id)
	if i < 0 {
		return res, ErrNotFound
	}
	c := lib.Concepts[i]
	res.Deleted = append(res.Deleted, id)

	if withReverse {
		for _, revID := range reverseIDs(lib, c) {
			r := &lib.Concepts[lib.Index(revID)]
			if remaining := slices.DeleteFunc(slices.Clone(r.TargetWordIDs), func(w string) bool { return w == c.SourceWordID }); len(remaining) > 0 {
				r.TargetWordIDs = remaining
				res.Updated = append(res.Updated, revID)
			} else {
				res.Deleted = append(res.Deleted, revID)
			}
		}
	}

	lib.Concepts = slices.DeleteFunc(lib.Concepts, func(x storage.TranslationConcept) bool { return slices.Contains(res.Deleted, x.ID) })
	removeUnusedWords(lib)
	return res, nil
}

func removeUnusedWords(lib *storage.Library) {
	used := map[string]bool{}
	for _, c := range lib.Concepts {
		used[c.SourceWordID] = true
		for _, t := range c.TargetWordIDs {
			used[t] = true
		}
	}
	for id := range lib.Words {
		if !used[id] {
			delete(lib.Words, id)
		}
	}
}
