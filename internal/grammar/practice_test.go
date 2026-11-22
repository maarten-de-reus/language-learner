package grammar

import (
	"math/rand/v2"
	"testing"
	"time"

	"hungarian-language-learner/internal/storage"
)

func practiceLib() *storage.Library {
	lib := &storage.Library{Words: map[string]storage.Word{
		"nl_huis": {ID: "nl_huis", Language: "nl", Text: "huis"}, "hu_haz": {ID: "hu_haz", Language: "hu", Text: "ház"},
		"nl_leest": {ID: "nl_leest", Language: "nl", Text: "leest"}, "hu_olvas": {ID: "hu_olvas", Language: "hu", Text: "olvas"},
		"nl_lezen": {ID: "nl_lezen", Language: "nl", Text: "lezen"}, "hu_olvasni": {ID: "hu_olvasni", Language: "hu", Text: "olvasni"},
		"nl_tuin": {ID: "nl_tuin", Language: "nl", Text: "tuin"}, "hu_kert": {ID: "hu_kert", Language: "hu", Text: "kert"},
	}}
	lib.Concepts = []storage.TranslationConcept{
		{ID: "a", SourceWordID: "nl_huis", TargetWordIDs: []string{"hu_haz"}, WordType: "zn", FirstReview: time.Now()},
		{ID: "b", SourceWordID: "hu_haz", TargetWordIDs: []string{"nl_huis"}, WordType: "zn"},
		{ID: "c", SourceWordID: "nl_leest", TargetWordIDs: []string{"hu_olvas"}, WordType: "ww"},
		{ID: "d", SourceWordID: "nl_lezen", TargetWordIDs: []string{"hu_olvasni"}, WordType: "ww"},
		{ID: "e", SourceWordID: "nl_tuin", TargetWordIDs: []string{"hu_kert"}},
	}
	return lib
}

// fewWords: tests met een paar woorden; een regel heeft er normaal MinWords nodig
func fewWords(t *testing.T) {
	old := MinWords
	MinWords = 1
	t.Cleanup(func() { MinWords = old })
}

func TestWordsAndNext(t *testing.T) {
	fewWords(t)
	words := Words(practiceLib())
	if len(words) != 2 || words[0].Hungarian != "ház" || !words[0].Known || words[1].Hungarian != "olvas" {
		t.Fatalf("woorden: %+v (verwacht ház en olvas: geen infinitief, geen woord zonder woordsoort, HU->NL niet dubbel)", words)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	seen := map[string]bool{}
	for range 50 {
		ex, ok := Next(words, []string{"ban", "ww-onbep"}, nil, "", rng)
		if !ok {
			t.Fatal("verwacht een opgave")
		}
		seen[ex.Rule+":"+ex.Word] = true
		if ex.Rule == "ww-onbep" && (ex.PersonLabel == "" || ex.Person == int(Sg3)) {
			t.Errorf("werkwoordopgave zonder persoon of met de 3e persoon (staat al op de kaart): %+v", ex)
		}
	}
	if !seen["ban:ház"] || !seen["ww-onbep:olvas"] || len(seen) != 2 {
		t.Errorf("opgaven: %v", seen)
	}
	if _, ok := Next(words, []string{"ww-bep"}[:0], nil, "", rng); ok {
		t.Error("zonder actieve regels geen opgave")
	}
}

func TestCheckAndScores(t *testing.T) {
	if r, _ := Check("ban", Word{Hungarian: "ház"}, 0, " Házban "); !r.Correct {
		t.Errorf("házban moet goed zijn: %+v", r)
	}
	if r, _ := Check("ban", Word{Hungarian: "ház"}, 0, "hazban"); r.Correct || !r.AccentOnly || r.Form.Text != "házban" {
		t.Errorf("hazban: alleen accenten fout: %+v", r)
	}
	if r, _ := Check("ww-onbep", Word{Hungarian: "olvas"}, int(Sg2), "olvassz"); r.Correct || r.Form.Text != "olvasol" || len(r.Form.Steps) == 0 {
		t.Errorf("olvas, te: %+v", r)
	}
	if _, ok := Check("bestaat-niet", Word{Hungarian: "ház"}, 0, "x"); ok {
		t.Error("onbekende regel")
	}
	var log []storage.GrammarAttempt
	for i := range 25 {
		log = append(log, storage.GrammarAttempt{Rule: "ban", Correct: i >= 10})
	}
	s := Scores(log)["ban"]
	if s.Attempts != 25 || s.Correct != 15 || s.RecentN != 20 || s.RecentCorrect != 15 {
		t.Errorf("score: %+v", s)
	}
}

func TestExamplesVary(t *testing.T) {
	words := []Word{{Hungarian: "ház", Type: "zn", Known: true}, {Hungarian: "bolt", Type: "zn", Known: true}, {Hungarian: "kert", Type: "zn"}}
	r, _ := RuleByID("ban")
	ex := r.Examples(words, 2)
	if len(ex) != 2 || ex[0].Form != "házban" || ex[1].Form != "kertben" {
		t.Errorf("voorbeelden moeten -ban én -ben laten zien: %+v", ex)
	}
}

func TestOverrides(t *testing.T) {
	forms, err := ParseForms("t=szekeret, k=szekerek, on=?, te=mész")
	if err != nil {
		t.Fatal(err)
	}
	w := Word{Hungarian: "szekér", Type: "zn", Forms: forms}
	if r, _ := Check("t", w, 0, "szekeret"); !r.Correct {
		t.Errorf("afwijkende vorm moet voorgaan: %+v", r)
	}
	on, _ := RuleByID("on")
	if len(on.persons(w)) != 0 {
		t.Error("on=? betekent: niet oefenen")
	}
	if r, _ := Check("ban", w, 0, "szekérben"); !r.Correct {
		t.Errorf("zonder afwijking gewoon de regel: %+v", r)
	}
	v := Word{Hungarian: "xmegy", Type: "ww", Forms: forms}
	if r, _ := Check("ww-onbep", v, int(Sg2), "mész"); !r.Correct {
		t.Errorf("persoon te: %+v", r)
	}
	if FormatForms(forms) != "k=szekerek, on=?, t=szekeret, te=mész" {
		t.Errorf("tekst: %q", FormatForms(forms))
	}
	noDef, _ := ParseForms("bep=?")
	if bep, _ := RuleByID("ww-bep"); len(bep.persons(Word{Hungarian: "olvas", Forms: noDef})) != 0 {
		t.Error("bep=? betekent: geen bepaalde vervoeging")
	}
	for _, bad := range []string{"t", "x=y", "t=twee woorden", "bep=olvasom"} {
		if _, err := ParseForms(bad); err == nil {
			t.Errorf("%q moet een fout geven", bad)
		}
	}
}

func TestIRuleUsesPlaces(t *testing.T) {
	r, _ := RuleByID("i")
	own := []Word{{Hungarian: "kávé", Type: "zn", Known: true}, {Hungarian: "reggel", Dutch: "ochtend (mijn kaart)", Type: "zn", Known: true}}
	pool := r.Applicable(own)
	for _, w := range pool {
		if w.Hungarian == "kávé" {
			t.Error("kávé hoort niet bij de -i-regel (kávéi = zijn koffies)")
		}
		if w.Hungarian == "reggel" && (!w.Known || w.Dutch != "ochtend") {
			t.Errorf("eigen woord moet als bekend tellen: %+v", w)
		}
	}
	for word, want := range map[string]string{"Budapest": "budapesti", "este": "esti", "hónap": "havi", "Eger": "egri", "Hollandia": "hollandiai", "szerda": "szerdai"} {
		if res, ok := Check("i", r.Lookup(nil, word), 0, want); !ok || !res.Correct {
			t.Errorf("%s -> %s: %+v", word, want, res)
		}
	}
}

func TestOnlyKnownWords(t *testing.T) {
	var words []Word
	for i, hu := range []string{"ház", "kert", "bolt", "asztal", "szék", "ablak"} {
		words = append(words, Word{Hungarian: hu, Type: "zn", Step: i}) // stap 0 t/m 5
	}
	known := KnownWords(words, 2)
	if len(known) != 4 || known[0].Hungarian != "bolt" {
		t.Fatalf("vanaf stap 2: %+v", known)
	}
	ban, _ := RuleByID("ban")
	if n, ready := ban.Ready(known); n != 4 || ready {
		t.Errorf("4 bekende woorden, er zijn er %d nodig: n=%d ready=%v", MinWords, n, ready)
	}
	if n, ready := ban.Ready(KnownWords(words, 1)); n != 5 || !ready {
		t.Errorf("vanaf stap 1: n=%d ready=%v", n, ready)
	}
	rng := rand.New(rand.NewPCG(3, 4))
	if _, ok := Next(known, []string{"ban"}, nil, "", rng); ok {
		t.Error("een regel die nog niet klaar is geeft geen opgave")
	}
	if ex, ok := Next(known, []string{"ban", "i"}, nil, "", rng); !ok || ex.Rule != "i" {
		t.Errorf("de -i-regel (vaste lijst) is altijd klaar: %+v %v", ex, ok)
	}
	w := Words(practiceLib())
	if w[0].Hungarian != "ház" || w[0].Step != 0 || w[1].Step != -1 {
		t.Errorf("stap per woord: %+v", w)
	}
}
