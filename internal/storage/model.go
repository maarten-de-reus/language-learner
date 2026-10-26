package storage

import "time"

// Word representeert één los bouwsteen-woord in een specifieke taal
type Word struct {
	ID       string `json:"id"`
	Language string `json:"language"` // "nl", "hu", "en"
	Text     string `json:"text"`     // "bank", "pad", "kanapé"
}

// TranslationConcept is de 'lijm' die per leerrichting wordt aangemaakt
type TranslationConcept struct {
	ID            string   `json:"id"`
	SourceWordID  string   `json:"source_word_id"`         // Verwijst naar bijv. "bank" (NL)
	ContextSource string   `json:"context_source"`         // Context in taal van de bron: "Zitmeubilair"
	TargetWordIDs []string `json:"target_word_ids"`        // Slice van geaccepteerde synoniem-IDs ["w_pad", "w_kanape"]
	Example       string   `json:"example,omitempty"`      // Voorbeeldzin, bijv. "Milyen lesz az idő holnap?"
	ExampleLang   string   `json:"example_lang,omitempty"` // Taal van de voorbeeldzin, bijv. "hu"
	Groups        []string `json:"groups,omitempty"`       // Groepen waar het woord in zit, bijv. ["Getallen", "Week 1"]
	// Voortgang: wordt in voortgang.json bewaard (zie ProgressState), nooit in woorden.json
	StepIndex  int       `json:"-"` // Index voor engine.AnkiIntervals (0 t/m 5)
	NextReview time.Time `json:"-"`
	// Moment van het eerste antwoord; leeg betekent een nieuw woord dat nog in de wachtrij staat
	FirstReview time.Time `json:"-"`
	// Moment van het laatste echte antwoord (geen oefenherhaling); de andere richting wacht dan tot morgen
	LastReview time.Time `json:"-"`
}
