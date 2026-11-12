package catalog

import (
	"strings"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

// Review legt de controle van een woord vast, voor het concept en zijn omgekeerde richting(en):
// goedgekeurd (een eerdere opmerking vervalt), of niet goedgekeurd met een opmerking voor de
// leerling. Pas na goedkeuring komt een nieuw woord in de les. Geeft de aangepaste concepten.
func Review(lib *storage.Library, id string, verified bool, note string) ([]string, error) {
	i := lib.Index(id)
	if i < 0 {
		return nil, ErrNotFound
	}
	ids := append([]string{id}, reverseIDs(lib, lib.Concepts[i])...)
	note = strings.TrimSpace(note)
	for _, cid := range ids {
		c := &lib.Concepts[lib.Index(cid)]
		c.Verified = verified
		c.ReviewNote = note
		if verified {
			c.ReviewNote = ""
		}
	}
	return ids, nil
}

// SetPriority geeft een nieuw woord (in beide richtingen) voorrang in de wachtrij, of haalt die
// weg. Woorden die al in de herhalingen zitten blijven ongemoeid. Geeft de aangepaste concepten.
func SetPriority(lib *storage.Library, id string, on bool) ([]string, error) {
	i := lib.Index(id)
	if i < 0 {
		return nil, ErrNotFound
	}
	var changed []string
	for _, cid := range append([]string{id}, reverseIDs(lib, lib.Concepts[i])...) {
		if c := &lib.Concepts[lib.Index(cid)]; engine.IsNew(*c) {
			c.Priority = on
			changed = append(changed, cid)
		}
	}
	return changed, nil
}
