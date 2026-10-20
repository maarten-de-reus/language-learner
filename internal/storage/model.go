package storage

import "time"

// Progress houdt de Spaced Repetition status bij per unieke leerrichting
type Progress struct {
	NextReview time.Time `json:"next_review"`
	StepIndex  int       `json:"step_index"` // Index voor de AnkiIntervals (0 t/m 5)
}

// Word representeert één los bouwsteen-woord in een specifieke taal
type Word struct {
	ID       string `json:"id"`
	Language string `json:"language"` // "nl", "hu", "en"
	Text     string `json:"text"`     // "bank", "pad", "kanapé"
}

// TranslationConcept is de 'lijm' die per leerrichting wordt aangemaakt
type TranslationConcept struct {
	ID            string    `json:"id"`
	SourceWordID  string    `json:"source_word_id"` // Verwijst naar bijv. "bank" (NL)
	ContextSource string    `json:"context_source"` // Context in taal van de bron: "Zitmeubilair"
	TargetWordIDs []string  `json:"target_word_ids"` // Slice van geaccepteerde synoniem-IDs ["w_pad", "w_kanape"]
	StepIndex     int       `json:"step_index"`
	NextReview    time.Time `json:"next_review"`
}