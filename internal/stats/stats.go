// Package stats berekent statistieken uit het antwoordenlog en de huidige voortgang.
// Alles wordt telkens opnieuw uitgerekend: het log is klein en zo blijft er geen tweede
// bron van waarheid om bij te houden. Dagen volgen dezelfde grens als de planning (04:00).
package stats

import (
	"slices"
	"sort"
	"strings"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

const (
	ActivityDays  = 30 // dagen in de activiteitsgrafiek
	ForecastDays  = 14 // dagen in de vooruitblik
	HardestLimit  = 10 // aantal moeilijkste woorden
	MinRetryCases = 5  // minimaal aantal gevallen voordat het effect van herhalen iets zegt
	MinLapseStep  = 2  // een woord telt als vergeten vanaf deze stap (7 dagen): daarvoor ben je het nog aan het leren
)

type Report struct {
	Answers        int     `json:"answers"`       // alle antwoorden, inclusief oefenherhalingen
	FirstAttempts  int     `json:"firstAttempts"` // antwoorden die meetellen voor de planning
	FirstCorrect   int     `json:"firstCorrect"`
	Accuracy       float64 `json:"accuracy"` // FirstCorrect / FirstAttempts, 0..1
	WordsPracticed int     `json:"wordsPracticed"`
	DaysPracticed  int     `json:"daysPracticed"`
	Streak         int     `json:"streak"` // aaneengesloten dagen tot vandaag (of gisteren, als je vandaag nog niet hebt geoefend)
	LongestStreak  int     `json:"longestStreak"`
	PracticedToday bool    `json:"practicedToday"`
	Today          Day     `json:"today"`

	Mistakes   Mistakes    `json:"mistakes"`
	Directions []Direction `json:"directions"` // per leerrichting, bijv. hu -> nl en nl -> hu
	Activity   []Day       `json:"activity"`   // laatste ActivityDays dagen, oudste eerst
	Forecast   []DueDay    `json:"forecast"`   // vandaag (inclusief achterstand) en de dagen erna
	Hardest    []Hard      `json:"hardest"`
	Retry      RetryEffect `json:"retry"`
	Lapses     Lapses      `json:"lapses"`
	Queue      Queue       `json:"queue"`
	Skipped    int         `json:"skipped,omitempty"` // onleesbare regels in het log
}

type Day struct {
	Date    string `json:"date"` // JJJJ-MM-DD
	Correct int    `json:"correct"`
	Wrong   int    `json:"wrong"`
	Retries int    `json:"retries"`
}

type DueDay struct {
	Date string `json:"date"`
	Due  int    `json:"due"`
}

// Direction is de score van één leerrichting. Het allereerste antwoord op een woord is vaak
// een gok (zeker naar het Hongaars) en staat daarom apart van de echte herhalingen.
type Direction struct {
	From   string `json:"from"`
	To     string `json:"to"`
	New    Score  `json:"new"`    // eerste keer gevraagd
	Review Score  `json:"review"` // later gevraagd: wat je onthouden hebt
}

type Score struct {
	Attempts int `json:"attempts"`
	Correct  int `json:"correct"`
}

type Mistakes struct {
	AccentOnly   int `json:"accentOnly"`
	OtherMeaning int `json:"otherMeaning"`
	Other        int `json:"other"`
}

// Hard is een concept dat je vaak fout hebt bij de eerste poging
type Hard struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Prompt    string    `json:"prompt"`
	Context   string    `json:"context"`
	Answers   []string  `json:"answers"`
	Attempts  int       `json:"attempts"`
	Wrong     int       `json:"wrong"`
	Step      int       `json:"step"`
	LastWrong time.Time `json:"lastWrong"`
}

// RetryEffect: van de keren dat je een woord fout had en het bij de oefenherhaling wél wist,
// hoe vaak wist je het bij de volgende echte toets (een latere dag) ook?
type RetryEffect struct {
	Cases       int  `json:"cases"`       // fout, daarna goed bij herhaling, en al opnieuw getoetst
	NextCorrect int  `json:"nextCorrect"` // waarvan de volgende echte toets goed
	Enough      bool `json:"enough"`      // genoeg gevallen om iets te zeggen
}

// Lapses volgt woorden die je al kende (minstens MinLapseStep) en toch fout had. Zo'n woord gaat
// engine.LapseSteps stappen terug en komt de volgende dag terug: wist je het dan weer, en bleef het
// daarna hangen op het lagere interval? Accentfouten tellen niet mee (die kosten maar één stap).
type Lapses struct {
	Cases       int  `json:"cases"`       // vergeten woorden
	Retested    int  `json:"retested"`    // waarvan al opnieuw gevraagd
	NextCorrect int  `json:"nextCorrect"` // waarvan die keer goed
	Kept        int  `json:"kept"`        // goed teruggekomen en daarna nog eens gevraagd, op het lagere interval
	KeptCorrect int  `json:"keptCorrect"` // waarvan goed
	Enough      bool `json:"enough"`      // genoeg opnieuw gevraagd om iets te zeggen
}

// Queue is de wachtrij met nieuwe woorden die nog nooit gevraagd zijn
type Queue struct {
	Waiting int `json:"waiting"`
	PerDay  int `json:"perDay"` // ingestelde limiet; 0 = geen limiet
	Days    int `json:"days"`   // dagen tot de wachtrij leeg is bij deze limiet (0 bij geen limiet of lege wachtrij)
}

// Counts zijn de tellingen per concept, ook gebruikt in het beheer
type Counts struct {
	Attempts int // eerste pogingen
	Wrong    int
}

// PerConcept telt per concept de eerste pogingen en hoe vaak die fout waren
func PerConcept(reviews []storage.Review) map[string]Counts {
	out := map[string]Counts{}
	for _, r := range reviews {
		if r.Retry {
			continue
		}
		c := out[r.ConceptID]
		c.Attempts++
		if !r.Correct {
			c.Wrong++
		}
		out[r.ConceptID] = c
	}
	return out
}

// Compute maakt het volledige rapport. now bepaalt wat "vandaag" is; newPerDay is de
// dagelijkse limiet voor nieuwe woorden (0 = geen limiet).
func Compute(lib *storage.Library, reviews []storage.Review, now time.Time, newPerDay int) Report {
	reviews = slices.Clone(reviews)
	sort.SliceStable(reviews, func(i, j int) bool { return reviews[i].Time.Before(reviews[j].Time) })

	today := engine.DayOf(now)
	var rep Report
	byDay := map[string]*Day{}
	words := map[string]bool{}

	for _, r := range reviews {
		rep.Answers++
		day := engine.DayOf(r.Time.In(now.Location())).Format(time.DateOnly)
		d := byDay[day]
		if d == nil {
			d = &Day{Date: day}
			byDay[day] = d
		}
		if r.Retry {
			d.Retries++
			continue
		}
		words[r.ConceptID] = true
		rep.FirstAttempts++
		if r.Correct {
			rep.FirstCorrect++
			d.Correct++
			continue
		}
		d.Wrong++
		switch {
		case r.AccentOnly:
			rep.Mistakes.AccentOnly++
		case r.OtherMeaning:
			rep.Mistakes.OtherMeaning++
		default:
			rep.Mistakes.Other++
		}
	}
	if rep.FirstAttempts > 0 {
		rep.Accuracy = float64(rep.FirstCorrect) / float64(rep.FirstAttempts)
	}
	rep.WordsPracticed = len(words)
	rep.DaysPracticed = len(byDay)
	if d := byDay[today.Format(time.DateOnly)]; d != nil {
		rep.Today = *d
		rep.PracticedToday = true
	} else {
		rep.Today = Day{Date: today.Format(time.DateOnly)}
	}

	rep.Streak, rep.LongestStreak = streaks(byDay, today)
	for i := ActivityDays - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i).Format(time.DateOnly)
		if d := byDay[date]; d != nil {
			rep.Activity = append(rep.Activity, *d)
		} else {
			rep.Activity = append(rep.Activity, Day{Date: date})
		}
	}
	rep.Forecast = forecast(lib, today)
	rep.Directions = directions(lib, reviews)
	rep.Hardest = hardest(lib, reviews)
	rep.Retry = retryEffect(reviews, now.Location())
	rep.Lapses = lapses(reviews, now.Location())
	rep.Queue = Queue{PerDay: newPerDay}
	for _, c := range lib.Concepts {
		if engine.IsNew(c) {
			rep.Queue.Waiting++
		}
	}
	if newPerDay > 0 {
		rep.Queue.Days = (rep.Queue.Waiting + newPerDay - 1) / newPerDay
	}
	return rep
}

// streaks telt de huidige reeks (eindigend vandaag, of gisteren als je vandaag nog niet oefende)
// en de langste reeks ooit
func streaks(byDay map[string]*Day, today time.Time) (current, longest int) {
	var days []time.Time
	for date := range byDay {
		d, _ := time.ParseInLocation(time.DateOnly, date, today.Location())
		days = append(days, d)
	}
	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })

	run := 0
	for i, d := range days {
		if i > 0 && engine.DaysBetween(days[i-1], d) == 1 {
			run++
		} else {
			run = 1
		}
		longest = max(longest, run)
	}
	if len(days) > 0 {
		if gap := engine.DaysBetween(days[len(days)-1], today); gap <= 1 {
			current = run
		}
	}
	return current, longest
}

// forecast telt per dag hoeveel concepten aan de beurt zijn; wat al te laat is telt bij vandaag
func forecast(lib *storage.Library, today time.Time) []DueDay {
	out := make([]DueDay, ForecastDays)
	for i := range out {
		out[i].Date = today.AddDate(0, 0, i).Format(time.DateOnly)
	}
	for _, c := range lib.Concepts {
		if engine.IsNew(c) {
			continue // nieuwe woorden staan in de wachtrij, nog niet ingepland
		}
		i := max(0, engine.DaysBetween(today, engine.DayOf(c.NextReview.In(today.Location()))))
		if i < ForecastDays {
			out[i].Due++
		}
	}
	return out
}

// directions telt de eerste pogingen per leerrichting; verwijderde concepten tellen niet mee
func directions(lib *storage.Library, reviews []storage.Review) []Direction {
	byKey := map[string]*Direction{}
	seen := map[string]bool{}
	for _, r := range reviews {
		if r.Retry {
			continue
		}
		first := !seen[r.ConceptID]
		seen[r.ConceptID] = true
		i := lib.Index(r.ConceptID)
		if i < 0 || len(lib.Concepts[i].TargetWordIDs) == 0 {
			continue
		}
		c := lib.Concepts[i]
		from, to := lib.Words[c.SourceWordID].Language, lib.Words[c.TargetWordIDs[0]].Language
		d := byKey[from+"-"+to]
		if d == nil {
			d = &Direction{From: from, To: to}
			byKey[from+"-"+to] = d
		}
		score := &d.Review
		if first {
			score = &d.New
		}
		score.Attempts++
		if r.Correct {
			score.Correct++
		}
	}
	out := []Direction{}
	for _, d := range byKey {
		out = append(out, *d)
	}
	// Vaste volgorde, op brontaal: hu -> nl (herkennen) vóór nl -> hu (zelf bedenken)
	slices.SortFunc(out, func(a, b Direction) int { return strings.Compare(a.From, b.From) })
	return out
}

// hardest geeft de concepten die bij de eerste poging het vaakst fout gingen
func hardest(lib *storage.Library, reviews []storage.Review) []Hard {
	byID := map[string]*Hard{}
	for _, r := range reviews {
		if r.Retry {
			continue
		}
		h := byID[r.ConceptID]
		if h == nil {
			h = &Hard{ID: r.ConceptID}
			byID[r.ConceptID] = h
		}
		h.Attempts++
		if !r.Correct {
			h.Wrong++
			h.LastWrong = r.Time
		}
	}

	var out []Hard
	for _, h := range byID {
		i := lib.Index(h.ID)
		if h.Wrong == 0 || i < 0 { // verwijderde concepten tellen niet mee
			continue
		}
		c := lib.Concepts[i]
		src := lib.Words[c.SourceWordID]
		h.From, h.Prompt, h.Context, h.Step = src.Language, src.Text, c.ContextSource, c.StepIndex
		for _, t := range c.TargetWordIDs {
			h.Answers = append(h.Answers, lib.Words[t].Text)
			h.To = lib.Words[t].Language
		}
		out = append(out, *h)
	}
	// Meeste fouten eerst; bij gelijkspel het laagste percentage goed, dan de meest recente fout
	slices.SortFunc(out, func(a, b Hard) int {
		if a.Wrong != b.Wrong {
			return b.Wrong - a.Wrong
		}
		if ra, rb := float64(a.Wrong)/float64(a.Attempts), float64(b.Wrong)/float64(b.Attempts); ra != rb {
			if ra > rb {
				return -1
			}
			return 1
		}
		if c := b.LastWrong.Compare(a.LastWrong); c != 0 {
			return c
		}
		return strings.Compare(a.Prompt, b.Prompt)
	})
	if len(out) > HardestLimit {
		out = out[:HardestLimit]
	}
	return out
}

// retryEffect volgt per concept: eerste poging fout, daarna goed bij een oefenherhaling,
// en wat was de volgende echte toets (een eerste poging op een latere dag)?
func retryEffect(reviews []storage.Review, loc *time.Location) RetryEffect {
	var eff RetryEffect
	byConcept := map[string][]storage.Review{}
	for _, r := range reviews {
		byConcept[r.ConceptID] = append(byConcept[r.ConceptID], r)
	}
	for _, rs := range byConcept {
		for i, r := range rs {
			if r.Retry || r.Correct {
				continue
			}
			// Werd het in dezelfde run bij een herhaling alsnog goed?
			rescued := false
			j := i + 1
			for ; j < len(rs) && rs[j].Retry; j++ {
				rescued = rescued || rs[j].Correct
			}
			if !rescued || j >= len(rs) {
				continue
			}
			// rs[j] is de volgende echte toets; tel alleen als die op een latere dag was
			if engine.DaysBetween(engine.DayOf(r.Time.In(loc)), engine.DayOf(rs[j].Time.In(loc))) < 1 {
				continue
			}
			eff.Cases++
			if rs[j].Correct {
				eff.NextCorrect++
			}
		}
	}
	eff.Enough = eff.Cases >= MinRetryCases
	return eff
}

// lapses zoekt fouten volgens de huidige regel (StepAfter = StepBefore - LapseSteps; een volledige
// reset van vóór die regel telt niet mee) en volgt de twee echte toetsen op latere dagen
func lapses(reviews []storage.Review, loc *time.Location) Lapses {
	var l Lapses
	byConcept := map[string][]storage.Review{}
	for _, r := range reviews {
		if !r.Retry {
			byConcept[r.ConceptID] = append(byConcept[r.ConceptID], r)
		}
	}
	day := func(r storage.Review) time.Time { return engine.DayOf(r.Time.In(loc)) }
	// next geeft de eerste echte toets na rs[i] op een latere dag
	next := func(rs []storage.Review, i int) int {
		for j := i + 1; j < len(rs); j++ {
			if engine.DaysBetween(day(rs[i]), day(rs[j])) >= 1 {
				return j
			}
		}
		return -1
	}
	for _, rs := range byConcept {
		for i, r := range rs {
			if r.Correct || r.AccentOnly || r.StepBefore < MinLapseStep || r.StepAfter != max(0, r.StepBefore-engine.LapseSteps) {
				continue
			}
			l.Cases++
			j := next(rs, i)
			if j < 0 {
				continue
			}
			l.Retested++
			if !rs[j].Correct {
				continue
			}
			l.NextCorrect++
			if k := next(rs, j); k >= 0 {
				l.Kept++
				if rs[k].Correct {
					l.KeptCorrect++
				}
			}
		}
	}
	l.Enough = l.Retested >= MinRetryCases
	return l
}
