package engine

import (
	"math/rand/v2"
	"slices"
	"sort"
	"time"

	"hungarian-language-learner/internal/storage"
)

// AnkiIntervals zijn de intervallen in dagen per stap (StepIndex 0 t/m 5)
var AnkiIntervals = [6]int{1, 3, 7, 16, 40, 90}

// DayStartHour is het uur waarop een nieuwe herhaaldag begint (zoals in Anki). Wie om 01:00
// oefent, zit nog in de vorige dag. Het uur geldt in de tijdzone van de server (TZ).
const DayStartHour = 4

// ReviewDate geeft het begin van de herhaaldag die `days` dagen na de huidige herhaaldag valt.
// Zo wordt een woord dat je om 21:00 oefende morgenochtend herhaald, niet pas morgenavond.
func ReviewDate(now time.Time, days int) time.Time {
	day := now.Add(-DayStartHour * time.Hour)
	return time.Date(day.Year(), day.Month(), day.Day()+days, DayStartHour, 0, 0, 0, now.Location())
}

// DayOf geeft de herhaaldag waar een moment bij hoort (00:00 van die dag), met dezelfde
// grens van DayStartHour als de planning: 01:30 hoort nog bij de vorige dag
func DayOf(t time.Time) time.Time {
	d := t.Add(-DayStartHour * time.Hour)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, t.Location())
}

// DaysBetween telt het aantal kalenderdagen tussen twee dagen van DayOf (ook over zomer- en wintertijd)
func DaysBetween(from, to time.Time) int {
	utc := func(d time.Time) time.Time { return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC) }
	return int(utc(to).Sub(utc(from)).Hours() / 24)
}

// clampStep houdt een stap binnen de bekende intervallen, ook bij beschadigde voortgang
func clampStep(step int) int {
	return max(0, min(step, len(AnkiIntervals)-1))
}

// fuzzPercentage is de toegestane speling rond het basis-interval
func fuzzPercentage(baseDays int) float64 {
	switch {
	case baseDays < 7:
		return 0 // korte intervallen liggen vast
	case baseDays <= 16:
		return 0.10 // 10% voor 7 en 16 dagen
	default:
		return 0.05 // 5% voor 40 en 90 dagen
	}
}

// IntervalRange geeft het kortste en langste toegestane interval (in dagen) voor een stap
func IntervalRange(stepIndex int) (lo, hi int) {
	base := float64(AnkiIntervals[clampStep(stepIndex)])
	pct := fuzzPercentage(int(base))
	return int(base - base*pct + 0.5), int(base + base*pct + 0.5)
}

// Load geeft aan hoeveel concepten er op een herhaaldag (DayOf) al gepland staan
type Load func(day time.Time) int

// balancedInterval kiest binnen de toegestane speling de rustigste dag. Bij gelijke drukte
// wint het basis-interval; wat dan nog gelijk is, beslist het toeval (geen voorkeur voor korter
// of langer). Zonder Load wordt het een willekeurige dag, zoals voorheen.
func balancedInterval(stepIndex int, now time.Time, load Load) int {
	lo, hi := IntervalRange(stepIndex)
	if load == nil || lo == hi {
		return CalculateIntervalWithFuzz(stepIndex)
	}
	base := AnkiIntervals[clampStep(stepIndex)]
	days := make([]int, 0, hi-lo+1)
	for d := lo; d <= hi; d++ {
		days = append(days, d)
	}
	rand.Shuffle(len(days), func(i, j int) { days[i], days[j] = days[j], days[i] })

	best, bestScore := base, -1
	for _, d := range days {
		dist := d - base
		if dist < 0 {
			dist = -dist
		}
		// Drukte telt zwaar; afstand tot het basis-interval beslist bij gelijke drukte
		score := load(DayOf(ReviewDate(now, d)))*100 + dist
		if bestScore < 0 || score < bestScore {
			best, bestScore = d, score
		}
	}
	return best
}

// CalculateIntervalWithFuzz berekent de exacte speling op basis van het basis-interval
func CalculateIntervalWithFuzz(stepIndex int) int {
	baseDays := AnkiIntervals[clampStep(stepIndex)]
	maxPercentage := fuzzPercentage(baseDays)
	if maxPercentage == 0 {
		return baseDays
	}

	// Bereken random factor tussen -maxPercentage en +maxPercentage
	fuzzFactor := (rand.Float64() * (maxPercentage * 2)) - maxPercentage
	finalDays := float64(baseDays) + (float64(baseDays) * fuzzFactor)

	return int(finalDays + 0.5)
}

// UpdateProgressSmart verwerkt het antwoord en houdt rekening met vroegtijdig studeren.
// Met load wordt de volgende herhaling op de rustigste dag binnen de speling gepland.
func UpdateProgressSmart(concept *storage.TranslationConcept, isCorrect bool, load Load) {
	updateProgress(concept, isCorrect, time.Now(), load)
}

func updateProgress(concept *storage.TranslationConcept, isCorrect bool, now time.Time, load Load) {
	isEarly := now.Before(concept.NextReview)

	if !isCorrect {
		// Fout = Altijd harde reset, ook bij vroegtijdig oefenen
		concept.StepIndex = 0
		concept.NextReview = ReviewDate(now, 1)
		return
	}

	if isEarly {
		// Vroegtijdig GOED? Doe niks, timer en stap blijven ongewijzigd
		return
	}

	// Normale flow: Verhoog stap indien mogelijk
	concept.StepIndex = clampStep(concept.StepIndex + 1)
	concept.NextReview = ReviewDate(now, balancedInterval(concept.StepIndex, now, load))
}

// Unlimited als aantal nieuwe woorden: geen limiet
const Unlimited = -1

// IsNew is true voor een concept dat nog nooit gevraagd is (het staat in de wachtrij)
func IsNew(c storage.TranslationConcept) bool {
	return c.FirstReview.IsZero()
}

// GetNextChunk haalt de eerstvolgende 'run' op en geeft aan hoeveel er overblijven.
// Herhalingen van bekende woorden zijn altijd welkom; nieuwe woorden alleen tot newAllowance
// (Unlimited = geen limiet), in de volgorde waarin ze zijn toegevoegd.
// Laagste stappen (meest kritiek) gaan voor, binnen dezelfde stap in willekeurige volgorde.
// Twee richtingen van hetzelfde woord komen nooit in dezelfde run: het antwoord van de ene
// is de vraag van de andere. De tweede richting wacht op een volgende run.
func GetNextChunk(allConcepts []storage.TranslationConcept, chunkSize, newAllowance int) (currentChunk []storage.TranslationConcept, remainingCount int) {
	var dueWords, newWords []storage.TranslationConcept
	now := time.Now()

	for _, c := range allConcepts {
		if !c.NextReview.Before(now) {
			continue
		}
		if IsNew(c) {
			newWords = append(newWords, c)
		} else {
			dueWords = append(dueWords, c)
		}
	}
	if newAllowance != Unlimited && len(newWords) > newAllowance {
		newWords = newWords[:max(0, newAllowance)]
	}
	dueWords = append(dueWords, newWords...)

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
