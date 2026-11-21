// De regeloefening: een opgave maken uit je eigen woorden, een antwoord nakijken en de score per
// regel bijhouden. Los van de herhalingen: wat je hier antwoordt verandert niets aan je planning.

package grammar

import (
	"math/rand/v2"
	"slices"
	"sort"
	"strings"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"
)

// DefaultActive is de keuze zolang je zelf nog niets gekozen hebt
var DefaultActive = []string{"ban", "ba", "bol", "ww-onbep", "ww-inf"}

// Word is een woord uit je lijst waarop regels kunnen werken
type Word struct {
	Hungarian string
	Dutch     string
	Type      string
	Known     bool              // al eens geoefend in de herhalingen
	Forms     map[string]string // afwijkende vormen bij het woord (zie ParseForms)
}

// Words haalt de bruikbare woorden uit de woordenschat: per Hongaars woord één keer, met de
// woordsoort van zijn kaart. Werkwoorden alleen in de 3e persoon (olvas, niet olvasni): daaruit
// maken de regels de andere vormen.
func Words(lib *storage.Library) []Word {
	seen := map[string]int{}
	var out []Word
	for _, c := range lib.Concepts {
		if c.WordType == "" || len(c.TargetWordIDs) == 0 || lib.Words[c.SourceWordID].Language != "nl" {
			continue
		}
		dutch := lib.Words[c.SourceWordID].Text
		if hint := meaningHint(c.ContextSource); hint != "" {
			dutch += " (" + hint + ")"
		}
		for _, id := range c.TargetWordIDs {
			hu := strings.TrimSpace(lib.Words[id].Text)
			if hu == "" || strings.ContainsAny(hu, " -?!.") || (c.WordType == "ww" && strings.HasSuffix(hu, "ni")) {
				continue
			}
			key := strings.ToLower(hu) + "|" + c.WordType
			if i, ok := seen[key]; ok {
				out[i].Known = out[i].Known || !c.FirstReview.IsZero()
				if out[i].Forms == nil {
					out[i].Forms = c.Forms
				}
				continue
			}
			seen[key] = len(out)
			out = append(out, Word{Hungarian: hu, Dutch: dutch, Type: c.WordType, Known: !c.FirstReview.IsZero(), Forms: c.Forms})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return engine.Fold(out[i].Hungarian) < engine.Fold(out[j].Hungarian) })
	return out
}

// meaningHint laat uit de hint weg wat de woordsoort al zegt: "werkwoord · 3e persoon
// enkelvoud · vuur maken" wordt "vuur maken"
func meaningHint(hint string) string {
	var keep []string
	for _, part := range strings.Split(hint, "·") {
		p := strings.TrimSpace(part)
		l := strings.ToLower(p)
		if p == "" || l == "werkwoord" || l == "infinitief" || strings.HasPrefix(l, "3e persoon") ||
			l == "zelfstandig naamwoord" || l == "bijvoeglijk naamwoord" {
			continue
		}
		keep = append(keep, p)
	}
	return strings.Join(keep, " · ")
}

// persons geeft de personen waarvoor de regel een vorm heeft (bij gewone regels alleen Sg3)
func (r Rule) persons(w Word) []Person {
	if !r.Persons {
		if _, ok := r.MakeFor(w, Sg3); ok {
			return []Person{Sg3}
		}
		return nil
	}
	var ps []Person
	for p := Sg1; p <= Pl3; p++ {
		if r.ID == "ww-onbep" && p == Sg3 {
			continue // die staat al op je kaart
		}
		if _, ok := r.MakeFor(w, p); ok {
			ps = append(ps, p)
		}
	}
	return ps
}

// Applicable geeft de woorden waarop de regel werkt
func (r Rule) Applicable(words []Word) []Word {
	if r.pool != nil {
		var out []Word // een vaste lijst opgavewoorden (zoals plaatsen bij -i), niet al je woorden
		for _, w := range r.pool(words) {
			if len(r.persons(w)) > 0 {
				out = append(out, w)
			}
		}
		return out
	}
	var out []Word
	for _, w := range words {
		if w.Type == r.WordType && len(r.persons(w)) > 0 {
			out = append(out, w)
		}
	}
	return out
}

// Lookup zoekt het woord van een opgave op: in de vaste lijst van de regel als die er een heeft,
// anders in je eigen woorden (met zijn afwijkende vormen); onbekend geeft een kaal woord
func (r Rule) Lookup(words []Word, hungarian string) Word {
	if r.pool != nil {
		words = r.pool(words)
	}
	for _, w := range words {
		if w.Hungarian == hungarian {
			return w
		}
	}
	return Word{Hungarian: hungarian}
}

// Exercise is één opgave. Het antwoord gaat niet mee: dat rekent Check opnieuw uit.
type Exercise struct {
	Rule        string `json:"rule"`
	Word        string `json:"word"`
	Dutch       string `json:"dutch"`
	Person      int    `json:"person"`
	PersonLabel string `json:"personLabel,omitempty"`
}

// Next kiest een opgave: een actieve regel (regels met een lage score vaker), een woord dat je al
// kent (als dat kan) en bij werkwoorden een persoon. avoid is het woord van de vorige opgave.
func Next(words []Word, active []string, scores map[string]Score, avoid string, rng *rand.Rand) (Exercise, bool) {
	type option struct {
		rule  Rule
		words []Word
	}
	var opts []option
	total := 0.0
	weight := func(id string) float64 {
		s := scores[id]
		if s.RecentN == 0 {
			return 2 // nog nooit geoefend: wat vaker
		}
		return 0.5 + 2*(1-float64(s.RecentCorrect)/float64(s.RecentN))
	}
	for _, r := range Rules {
		if !slices.Contains(active, r.ID) {
			continue
		}
		if ws := r.Applicable(words); len(ws) > 0 {
			opts = append(opts, option{r, ws})
			total += weight(r.ID)
		}
	}
	if len(opts) == 0 {
		return Exercise{}, false
	}
	x := rng.Float64() * total
	o := opts[len(opts)-1]
	for _, cand := range opts {
		if x -= weight(cand.rule.ID); x < 0 {
			o = cand
			break
		}
	}
	pool := o.words
	var known []Word
	for _, w := range pool {
		if w.Known && w.Hungarian != avoid {
			known = append(known, w)
		}
	}
	if len(known) >= 5 && rng.Float64() < 0.8 {
		pool = known
	}
	w := pool[rng.IntN(len(pool))]
	if w.Hungarian == avoid && len(pool) > 1 {
		w = pool[(slices.IndexFunc(pool, func(x Word) bool { return x.Hungarian == avoid })+1)%len(pool)]
	}
	ps := o.rule.persons(w)
	p := ps[rng.IntN(len(ps))]
	ex := Exercise{Rule: o.rule.ID, Word: w.Hungarian, Dutch: w.Dutch, Person: int(p)}
	if o.rule.Persons {
		ex.PersonLabel = PersonLabels[p]
	}
	return ex, true
}

// Result is de uitkomst van Check
type Result struct {
	Correct    bool `json:"correct"`
	AccentOnly bool `json:"accentOnly,omitempty"` // fout, maar alleen in de accenten
	Form       Form `json:"form"`                 // de goede vorm met uitleg
}

// Check kijkt een antwoord na. ok is false als de regel of het woord niet (meer) bestaat.
func Check(ruleID string, word Word, person int, input string) (Result, bool) {
	r, ok := RuleByID(ruleID)
	if !ok || person < 0 || person > int(Pl3) {
		return Result{}, false
	}
	f, ok := r.MakeFor(word, Person(person))
	if !ok {
		return Result{}, false
	}
	in := norm(input)
	res := Result{Form: f}
	for _, want := range append([]string{f.Text}, f.Also...) {
		if in == norm(want) {
			res.Correct = true
		} else if engine.Fold(in) == engine.Fold(want) {
			res.AccentOnly = true
		}
	}
	if res.Correct {
		res.AccentOnly = false
	}
	return res, true
}

func norm(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.Trim(strings.TrimSpace(s), ".!?")), " "))
}

// Score per regel: alles, en de laatste RecentWindow antwoorden (zegt meer over nu)
type Score struct {
	Attempts      int `json:"attempts"`
	Correct       int `json:"correct"`
	RecentN       int `json:"recentN"`
	RecentCorrect int `json:"recentCorrect"`
}

const RecentWindow = 20

// Scores telt het regellog per regel
func Scores(log []storage.GrammarAttempt) map[string]Score {
	byRule := map[string][]bool{}
	for _, a := range log {
		byRule[a.Rule] = append(byRule[a.Rule], a.Correct)
	}
	out := map[string]Score{}
	for id, results := range byRule {
		var s Score
		for i, ok := range results {
			s.Attempts++
			recent := i >= len(results)-RecentWindow
			if recent {
				s.RecentN++
			}
			if ok {
				s.Correct++
				if recent {
					s.RecentCorrect++
				}
			}
		}
		out[id] = s
	}
	return out
}

// Example is een voorbeeld uit je eigen woorden
type Example struct {
	Word   string `json:"word"`
	Dutch  string `json:"dutch"`
	Person string `json:"person,omitempty"`
	Form   string `json:"form"`
}

// Examples geeft tot n voorbeelden uit je eigen woorden, liefst woorden die je al kent, met
// verschillende uitkomsten (bijv. -ban én -ben) als dat kan
func (r Rule) Examples(words []Word, n int) []Example {
	ws := r.Applicable(words)
	// Verspreid over je lijst (niet alleen woorden met een a), maar elke keer dezelfde
	rng := rand.New(rand.NewPCG(uint64(len(r.ID)), uint64(r.ID[0])))
	rng.Shuffle(len(ws), func(i, j int) { ws[i], ws[j] = ws[j], ws[i] })
	sort.SliceStable(ws, func(i, j int) bool { return ws[i].Known && !ws[j].Known })
	var out []Example
	endings := map[string]bool{}
	add := func(w Word, distinct bool) {
		if len(out) >= n || slices.ContainsFunc(out, func(e Example) bool { return e.Word == w.Hungarian }) {
			return
		}
		ps := r.persons(w)
		p := ps[len(out)%len(ps)]
		f, _ := r.MakeFor(w, p)
		end := ending(w.Hungarian, f.Text)
		if distinct && endings[end] {
			return
		}
		endings[end] = true
		e := Example{Word: w.Hungarian, Dutch: w.Dutch, Form: f.Text}
		if r.Persons {
			e.Person = PersonLabels[p]
		}
		out = append(out, e)
	}
	for _, w := range ws {
		add(w, true)
	}
	for _, w := range ws {
		add(w, false)
	}
	return out
}

// ending: wat er achter het woord kwam (ház -> házban: "ban"); voor de variatie in voorbeelden
func ending(word, form string) string {
	w, f := []rune(strings.ToLower(word)), []rune(strings.ToLower(form))
	i := 0
	for i < len(w) && i < len(f) && w[i] == f[i] {
		i++
	}
	return string(f[i:])
}
