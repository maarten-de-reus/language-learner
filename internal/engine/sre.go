package engine

import (
	"math/rand/v2"
	"slices"
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

// Outcome is de uitslag van een eerste poging
type Outcome int

const (
	Wrong      Outcome = iota
	AccentOnly         // goed woord, alleen de accenten fout
	Correct
)

// Na een fout komt een woord morgen terug, een paar stappen lager. Is het morgen goed, dan gaat het
// weer één stap omhoog: na een fout op 40 dagen volgen dus morgen, 16 en 40 dagen in plaats van
// opnieuw 1, 3, 7, 16 en 40. Een accentfout kost één stap: na morgen geldt weer het oude interval.
const (
	LapseSteps       = 2
	AccentLapseSteps = 1
)

// UpdateProgressSmart verwerkt het antwoord en houdt rekening met vroegtijdig studeren.
// Met load wordt de volgende herhaling op de rustigste dag binnen de speling gepland.
func UpdateProgressSmart(concept *storage.TranslationConcept, outcome Outcome, load Load) {
	updateProgress(concept, outcome, time.Now(), load)
}

func updateProgress(concept *storage.TranslationConcept, outcome Outcome, now time.Time, load Load) {
	isEarly := now.Before(concept.NextReview)

	if outcome != Correct {
		// Fout: stappen terug en morgen weer, ook bij vroegtijdig oefenen
		lapse := LapseSteps
		if outcome == AccentOnly {
			lapse = AccentLapseSteps
		}
		concept.StepIndex = clampStep(clampStep(concept.StepIndex) - lapse)
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

// Buried geeft de concepten die vandaag moeten wachten omdat een andere richting van hetzelfde
// woord vandaag al gevraagd is (in Anki: "bury siblings"). Wie net koffie -> kávé heeft gezien,
// heeft niets aan kávé -> koffie een paar minuten later; die komt morgen.
func Buried(all []storage.TranslationConcept, now time.Time) map[string]bool {
	today := DayOf(now)
	related := storage.NewReverses(all)
	buried := map[string]bool{}
	for _, r := range all {
		if r.LastReview.IsZero() || !DayOf(r.LastReview.In(now.Location())).Equal(today) {
			continue
		}
		// Ook kaarten die r maar van één kant raken: die verklappen het antwoord net zo goed
		for _, i := range related.Related(r) {
			buried[all[i].ID] = true
		}
	}
	return buried
}

// DueCandidates geeft alles wat vandaag gevraagd kan worden: herhalingen die aan de beurt zijn en
// nieuwe woorden tot newAllowance (Unlimited = geen limiet). Nieuwe woorden moeten goedgekeurd zijn
// (Verified) en komen in de volgorde waarin ze zijn toegevoegd, met voorrang (Priority) vooraan.
// Concepten waarvan vandaag al een andere richting gevraagd is, wachten tot morgen, en van een
// nieuw woord komt per dag één richting: "Nog 10" is zo ook echt 10 woorden.
func DueCandidates(all []storage.TranslationConcept, now time.Time, newAllowance int) []storage.TranslationConcept {
	buried := Buried(all, now)
	var due, queue []storage.TranslationConcept
	for _, c := range all {
		if !c.NextReview.Before(now) || buried[c.ID] {
			continue
		}
		switch {
		case !IsNew(c):
			due = append(due, c) // al in de herhalingen: daar is geen controle meer voor nodig
		case c.Verified:
			queue = append(queue, c)
		}
	}
	slices.SortStableFunc(queue, func(a, b storage.TranslationConcept) int {
		switch {
		case a.Priority && !b.Priority:
			return -1
		case b.Priority && !a.Priority:
			return 1
		}
		return 0
	})

	var fresh []storage.TranslationConcept
	for _, c := range queue {
		if newAllowance != Unlimited && len(fresh) >= newAllowance {
			break
		}
		if slices.ContainsFunc(fresh, func(o storage.TranslationConcept) bool { return storage.Related(c, o) }) {
			continue // andere richting van een woord dat vandaag al nieuw binnenkomt
		}
		fresh = append(fresh, c)
	}
	return append(due, fresh...)
}

// Mode bepaalt wat er in een ronde komt: herhalingen of nieuwe woorden, nooit door elkaar.
// Zo staan nieuwe woorden (bijv. een grote import) je herhalingen niet in de weg.
type Mode string

const (
	ModeReview Mode = "review" // woorden die je al eens gehad hebt en die aan de beurt zijn
	ModeNew    Mode = "new"    // woorden uit de wachtrij, tot de daglimiet
)

// Candidates geeft wat in deze modus vandaag gevraagd kan worden (zie DueCandidates)
func Candidates(all []storage.TranslationConcept, now time.Time, mode Mode, newAllowance int) []storage.TranslationConcept {
	if mode == ModeNew {
		return slices.DeleteFunc(DueCandidates(all, now, newAllowance), func(c storage.TranslationConcept) bool { return !IsNew(c) })
	}
	return DueCandidates(all, now, 0)
}

// NextChunk haalt de eerstvolgende run in deze modus op (zie GetNextChunk)
func NextChunk(all []storage.TranslationConcept, chunkSize int, mode Mode, newAllowance int) []storage.TranslationConcept {
	chunk, _ := chunkFrom(Candidates(all, time.Now(), mode, newAllowance), chunkSize)
	return chunk
}

// GetNextChunk haalt de eerstvolgende 'run' op en geeft aan hoeveel er vandaag nog overblijven,
// met herhalingen en nieuwe woorden door elkaar.
func GetNextChunk(allConcepts []storage.TranslationConcept, chunkSize, newAllowance int) (currentChunk []storage.TranslationConcept, remainingCount int) {
	return chunkFrom(DueCandidates(allConcepts, time.Now(), newAllowance), chunkSize)
}

// chunkFrom kiest een run uit de kandidaten, volledig willekeurig: korte en lange intervallen door
// elkaar, zodat je niet eerst alle woorden van 1 dag krijgt en daarna pas die van 3 en 7 dagen.
// Twee richtingen van hetzelfde woord komen nooit in dezelfde run, en (zie DueCandidates) ook
// niet op dezelfde dag.
func chunkFrom(dueWords []storage.TranslationConcept, chunkSize int) (currentChunk []storage.TranslationConcept, remainingCount int) {
	rand.Shuffle(len(dueWords), func(i, j int) { dueWords[i], dueWords[j] = dueWords[j], dueWords[i] })

	for _, c := range dueWords {
		if len(currentChunk) == chunkSize {
			break
		}
		if slices.ContainsFunc(currentChunk, func(other storage.TranslationConcept) bool { return storage.Related(c, other) }) {
			continue
		}
		currentChunk = append(currentChunk, c)
	}

	return currentChunk, len(dueWords) - len(currentChunk)
}
