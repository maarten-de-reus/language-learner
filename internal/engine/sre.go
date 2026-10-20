package engine

import (
	"math/rand/v2"
	"slices"
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

// GetNextChunk haalt de eerstvolgende 'run' op en geeft aan hoeveel er overblijven.
// Laagste stappen (meest kritiek) gaan voor, binnen dezelfde stap in willekeurige volgorde.
// Twee richtingen van hetzelfde woord komen nooit in dezelfde run: het antwoord van de ene
// is de vraag van de andere. De tweede richting wacht op een volgende run.
func GetNextChunk(allConcepts []storage.TranslationConcept, chunkSize int) (currentChunk []storage.TranslationConcept, remainingCount int) {
	var dueWords []storage.TranslationConcept
	now := time.Now()

	for _, c := range allConcepts {
		if c.NextReview.Before(now) {
			dueWords = append(dueWords, c)
		}
	}

	rand.Shuffle(len(dueWords), func(i, j int) { dueWords[i], dueWords[j] = dueWords[j], dueWords[i] })
	sort.SliceStable(dueWords, func(i, j int) bool {
		return dueWords[i].StepIndex < dueWords[j].StepIndex
	})

	for _, c := range dueWords {
		if len(currentChunk) == chunkSize {
			break
		}
		if slices.ContainsFunc(currentChunk, func(other storage.TranslationConcept) bool { return reverseOf(c, other) }) {
			continue
		}
		currentChunk = append(currentChunk, c)
	}

	// Ook de volgorde binnen de run husselen, zodat de meest kritieke niet altijd vooraan staan
	rand.Shuffle(len(currentChunk), func(i, j int) { currentChunk[i], currentChunk[j] = currentChunk[j], currentChunk[i] })
	return currentChunk, len(dueWords) - len(currentChunk)
}

// reverseOf is true als de twee concepten elkaars omgekeerde richting zijn (bijv. koffie -> kávé en kávé -> koffie)
func reverseOf(a, b storage.TranslationConcept) bool {
	return slices.Contains(a.TargetWordIDs, b.SourceWordID) || slices.Contains(b.TargetWordIDs, a.SourceWordID)
}
