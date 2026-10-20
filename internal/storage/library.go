package storage

import "slices"

// Library is de volledige woordenschat in het geheugen: woorden en concepten (inclusief voortgang).
// De quiz en het beheer werken op dezelfde Library, zodat een nieuw concept meteen geoefend kan worden.
// Niet veilig voor gelijktijdig gebruik; de aanroeper zorgt voor locking.
type Library struct {
	Words    map[string]Word
	Concepts []TranslationConcept
}

// LoadLibrary laadt woorden en voortgang via LoadData
func LoadLibrary(wordsFile, progressFile string) (*Library, error) {
	words, concepts, err := LoadData(wordsFile, progressFile)
	if err != nil {
		return nil, err
	}
	return &Library{Words: words, Concepts: concepts}, nil
}

// Index geeft de positie van een concept, of -1
func (l *Library) Index(id string) int {
	return slices.IndexFunc(l.Concepts, func(c TranslationConcept) bool { return c.ID == id })
}

// Clone maakt een diepe kopie, bijv. om een wijziging terug te draaien als opslaan mislukt
func (l *Library) Clone() *Library {
	c := &Library{Words: make(map[string]Word, len(l.Words)), Concepts: make([]TranslationConcept, len(l.Concepts))}
	for id, w := range l.Words {
		c.Words[id] = w
	}
	for i, concept := range l.Concepts {
		concept.TargetWordIDs = slices.Clone(concept.TargetWordIDs)
		c.Concepts[i] = concept
	}
	return c
}
