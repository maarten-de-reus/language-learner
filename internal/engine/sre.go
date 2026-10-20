package engine

import (
	"math/rand/v2"
	"sort"
	"time"

	"hungarian-language-learner/internal/storage"
)

// Constanten voor de Anki stappen via iota
const (
	Step1Day = iota
	Step3Days
	Step7Days
	Step16Days
	Step40Days
	Step90Days
)

// Rotsvast array in het geheugen voor maximale snelheid (geen slice overhead)
var AnkiIntervals = [6]int{1, 3, 7, 16, 40, 90}

// CalculateIntervalWithFuzz berekent de exacte speling op basis van het basis-interval
func CalculateIntervalWithFuzz(stepIndex int) int {
	baseDays := AnkiIntervals[stepIndex]
	if baseDays < 7 {
		return baseDays
	}

	var maxPercentage float64
	if baseDays <= 16 {
		maxPercentage = 0.10 // 10% fuzz voor 7 en 16 dagen
	} else {
		maxPercentage = 0.05 // 5% fuzz voor 40 en 90 dagen
	}

	// Bereken random factor tussen -maxPercentage en +maxPercentage
	fuzzFactor := (rand.Float64() * (maxPercentage * 2)) - maxPercentage
	finalDays := float64(baseDays) + (float64(baseDays) * fuzzFactor)

	return int(finalDays + 0.5)
}

// UpdateProgressSmart verwerkt het antwoord en houdt rekening met vroegtijdig studeren
func UpdateProgressSmart(concept *storage.TranslationConcept, isCorrect bool) {
	isEarly := time.Now().Before(concept.NextReview)

	if !isCorrect {
		// Fout = Altijd harde reset, ook bij vroegtijdig oefenen
		concept.StepIndex = 0
		concept.NextReview = time.Now().AddDate(0, 0, 1)
		return
	}

	if isEarly {
		// Vroegtijdig GOED? Doe niks, timer en stap blijven ongewijzigd
		return
	}

	// Normale flow: Verhoog stap indien mogelijk
	if concept.StepIndex < len(AnkiIntervals)-1 {
		concept.StepIndex++
	}

	actualDays := CalculateIntervalWithFuzz(concept.StepIndex)
	concept.NextReview = time.Now().AddDate(0, 0, actualDays)
}

// GetNextChunk haalt de eerstvolgende 'run' op en geeft aan hoeveel er overblijven
func GetNextChunk(allConcepts []storage.TranslationConcept, chunkSize int) (currentChunk []storage.TranslationConcept, remainingCount int) {
	var dueWords []storage.TranslationConcept
	now := time.Now()

	for _, c := range allConcepts {
		if c.NextReview.Before(now) {
			dueWords = append(dueWords, c)
		}
	}

	// Sorteer: Laagste stappen (meest kritiek) eerst
	sort.Slice(dueWords, func(i, j int) bool {
		return dueWords[i].StepIndex < dueWords[j].StepIndex
	})

	totalDue := len(dueWords)
	if totalDue <= chunkSize {
		return dueWords, 0
	}

	return dueWords[:chunkSize], totalDue - chunkSize
}
