// Antwoorden nakijken: in een ronde (Answer, Weet ik niet, Toch goed) en in Kaarten (Practice).

package quiz

import (
	"errors"
	"strings"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/grammar"
	"hungarian-language-learner/internal/storage"
)

// Feedback is de uitslag van het laatst gegeven antwoord
type Feedback struct {
	Correct    bool      `json:"correct"`
	AccentOnly bool      `json:"accentOnly"`
	Input      string    `json:"input"`
	Expected   []string  `json:"expected"`
	Step       int       `json:"step"`
	NextReview time.Time `json:"nextReview"`
	Retry      bool      `json:"retry"`      // dit was een oefenherhaling
	WillRetry  bool      `json:"willRetry"`  // het concept komt aan het eind van de run nog een keer
	Unknown    bool      `json:"unknown"`    // "Weet ik niet": geen antwoord gegeven
	Typo       bool      `json:"typo"`       // goed, maar met één tikfout (alleen naar het Nederlands)
	Overridden bool      `json:"overridden"` // fout nagekeken, maar met "Toch goed" alsnog goedgerekend
	Mastered   bool      `json:"mastered"`   // met dit antwoord beheerst geworden
	TopLeft    int       `json:"topLeft"`    // op 90 dagen: nog zoveel keer goed tot beheerst
	Example    string    `json:"example,omitempty"`
	// OtherMeaning is gevuld als het antwoord een goede vertaling is van een ándere betekenis
	// van hetzelfde woord (bijv. idő -> "tijd" terwijl "weer" bedoeld is). Het telt als fout.
	OtherMeaning *OtherMeaning `json:"otherMeaning,omitempty"`
	// Form is gevuld als het antwoord een andere vorm van het gevraagde woord is (olvasok bij
	// olvas): welke regel die vorm maakt. Het telt als fout.
	Form *grammar.Match `json:"form,omitempty"`
}

// OtherMeaning beschrijft de andere betekenis waar het antwoord bij hoort
type OtherMeaning struct {
	Answer  string `json:"answer"`
	Context string `json:"context"`
}

// Feedback geeft de uitslag van het huidige concept, of nil als het nog niet beantwoord is
func (s *Session) Feedback() *Feedback { return s.feedback }

// Answer kijkt het antwoord op het huidige concept na.
// Bij een eerste poging wordt de voortgang bijgewerkt en opgeslagen; bij een oefenherhaling niet.
// Een fout antwoord zet het concept (maximaal MaxRetries keer) achteraan de run.
// Geeft nil terug als er niets na te kijken valt (run klaar, of al beantwoord).
// Een fout betekent dat opslaan of het antwoordenlog mislukte; de voortgang is in het geheugen wel bijgewerkt.
func (s *Session) Answer(input string) (*Feedback, error) {
	return s.answer(input, false)
}

// DontKnow is "Weet ik niet": het telt als fout (ook voor de planning), maar in het log staat
// dat je het niet wist in plaats van een gok
func (s *Session) DontKnow() (*Feedback, error) {
	return s.answer("", true)
}

func (s *Session) answer(input string, unknown bool) (*Feedback, error) {
	if s.Done() || s.feedback != nil {
		return nil, nil
	}

	item := s.run[s.pos]
	concept := s.conceptOf(item)
	expected := s.targetTexts(concept)
	var correct, accentOnly, typo bool
	if !unknown {
		correct, accentOnly, typo = s.check(concept, input, expected)
	}
	stepBefore := concept.StepIndex
	s.undo = progress{step: concept.StepIndex, next: concept.NextReview, topStreak: concept.TopStreak}

	var saveErr error
	if item.retry == 0 {
		if engine.IsNew(*concept) {
			concept.FirstReview = time.Now() // uit de wachtrij: telt mee voor de daglimiet
		}
		concept.LastReview = time.Now() // de andere richting van dit woord wacht nu tot morgen
		outcome := engine.Wrong
		if correct {
			outcome = engine.Correct
		} else if accentOnly {
			outcome = engine.AccentOnly
		}
		engine.UpdateProgressSmart(concept, outcome, s.load(concept.ID))
		saveErr = s.Save()
		item.result = len(s.results)
		s.results = append(s.results, Result{
			Prompt:   s.lib.Words[concept.SourceWordID].Text,
			Input:    input,
			Expected: expected,
			Correct:  correct,
		})
	} else {
		r := &s.results[item.result]
		r.Retries = item.retry
		r.RetryCorrect = correct
	}

	willRetry := !correct && item.retry < MaxRetries
	if willRetry {
		s.run = append(s.run, runItem{conceptID: item.conceptID, retry: item.retry + 1, result: item.result})
	}

	s.feedback = &Feedback{
		Correct:    correct,
		AccentOnly: accentOnly,
		Input:      input,
		Expected:   expected,
		Step:       concept.StepIndex,
		NextReview: concept.NextReview,
		Retry:      item.retry > 0,
		WillRetry:  willRetry,
		Unknown:    unknown,
		Typo:       typo,
		Example:    concept.Example,
	}
	if !correct && !unknown {
		s.feedback.OtherMeaning = s.otherMeaning(concept, input)
		if s.feedback.OtherMeaning == nil && !accentOnly {
			s.feedback.Form = s.formOf(concept, input)
		}
	}
	if item.retry == 0 {
		s.mastery(item, stepBefore, concept)
	}
	return s.feedback, errors.Join(saveErr, s.logReview(concept, input, stepBefore))
}

// logReview schrijft het zojuist gegeven antwoord in het antwoordenlog (als dat ingesteld is)
func (s *Session) logReview(c *storage.TranslationConcept, input string, stepBefore int) error {
	if s.reviewLog == nil {
		return nil
	}
	fb := s.feedback
	return s.reviewLog.Append(storage.Review{
		Time:         time.Now(),
		ConceptID:    c.ID,
		Input:        input,
		Correct:      fb.Correct,
		AccentOnly:   fb.AccentOnly,
		OtherMeaning: fb.OtherMeaning != nil,
		Unknown:      fb.Unknown,
		Typo:         fb.Typo,
		Retry:        fb.Retry,
		StepBefore:   stepBefore,
		StepAfter:    c.StepIndex,
		NextReview:   c.NextReview,
	})
}

// PracticeResult is de uitslag van een oefenantwoord buiten de planning (Kaarten)
type PracticeResult struct {
	Correct      bool           `json:"correct"`
	AccentOnly   bool           `json:"accentOnly"`
	Typo         bool           `json:"typo"`
	Expected     []string       `json:"expected"`
	OtherMeaning *OtherMeaning  `json:"otherMeaning,omitempty"`
	Form         *grammar.Match `json:"form,omitempty"`
}

// Practice kijkt een antwoord na zoals Answer, maar verandert niets: geen voortgang, geen log
// en de lopende run blijft zoals hij is. ok is false als het concept niet bestaat.
func (s *Session) Practice(conceptID, input string) (res PracticeResult, ok bool) {
	i := s.lib.Index(conceptID)
	if i < 0 {
		return res, false
	}
	concept := &s.lib.Concepts[i]
	res.Expected = s.targetTexts(concept)
	res.Correct, res.AccentOnly, res.Typo = s.check(concept, input, res.Expected)
	if !res.Correct {
		res.OtherMeaning = s.otherMeaning(concept, input)
		if res.OtherMeaning == nil && !res.AccentOnly {
			res.Form = s.formOf(concept, input)
		}
	}
	return res, true
}

// check kijkt een antwoord na. Naar het Nederlands telt één tikfout als goed (typo): je kende het
// woord, je typte het alleen net verkeerd. Niet als je antwoord zelf een woord uit je lijst is
// (meer i.p.v. weer): dan bedoelde je waarschijnlijk dat andere woord. Naar het Hongaars niet:
// daar is de spelling juist wat je leert, en één letter is vaak een ander woord (ül / öl).
func (s *Session) check(c *storage.TranslationConcept, input string, expected []string) (correct, accentOnly, typo bool) {
	correct, accentOnly = engine.CheckAnswer(input, expected)
	if correct || accentOnly || len(c.TargetWordIDs) == 0 || s.lib.Words[c.TargetWordIDs[0]].Language != "nl" || s.isWord("nl", input) {
		return correct, accentOnly, false
	}
	if _, ok := engine.Typo(input, expected); ok {
		return true, false, true
	}
	return false, false, false
}

// isWord is true als text een woord in deze taal uit de woordenschat is
func (s *Session) isWord(lang, text string) bool {
	want := engine.Fold(text)
	for _, w := range s.lib.Words {
		if w.Language == lang && engine.Fold(w.Text) == want {
			return true
		}
	}
	return false
}

// Override is "Toch goed": het laatste foute antwoord telt alsnog als goed, bijv. een tikfout die
// de controle niet herkende of een synoniem dat niet in de lijst staat. Bij een eerste poging
// wordt de planning teruggedraaid en als goed verwerkt; de herhaling aan het eind vervalt.
// Geeft nil als er niets terug te draaien is (al goed, "Weet ik niet" of nog niet beantwoord).
func (s *Session) Override() (*Feedback, error) {
	fb := s.feedback
	if fb == nil || fb.Correct || fb.Unknown {
		return nil, nil
	}
	item := s.run[s.pos]
	concept := s.conceptOf(item)
	if concept == nil {
		return nil, nil
	}
	var saveErr error
	if item.retry == 0 {
		concept.StepIndex, concept.NextReview, concept.TopStreak = s.undo.step, s.undo.next, s.undo.topStreak
		engine.UpdateProgressSmart(concept, engine.Correct, s.load(concept.ID))
		saveErr = s.Save()
		s.results[item.result].Correct = true
	} else {
		s.results[item.result].RetryCorrect = true
	}
	if fb.WillRetry {
		s.run = s.run[:len(s.run)-1] // de herhaling die bij dit antwoord achteraan is gezet
	}
	fb.Correct, fb.AccentOnly, fb.OtherMeaning, fb.WillRetry, fb.Overridden = true, false, nil, false, true
	fb.Step, fb.NextReview = concept.StepIndex, concept.NextReview
	if item.retry == 0 {
		s.mastery(item, s.undo.step, concept)
	}

	var logErr error
	if s.reviewLog != nil {
		logErr = s.reviewLog.Append(storage.Review{
			Time: time.Now(), ConceptID: concept.ID, Input: fb.Input, Correct: true, Override: true, Retry: fb.Retry,
			StepBefore: s.undo.step, StepAfter: concept.StepIndex, NextReview: concept.NextReview,
		})
	}
	return fb, errors.Join(saveErr, logErr)
}

// mastery vult in wat een eerste poging betekende op weg naar Beheerst: net beheerst geworden,
// of nog zoveel keer goed nodig op 90 dagen
func (s *Session) mastery(item runItem, stepBefore int, c *storage.TranslationConcept) {
	fb := s.feedback
	fb.Mastered = stepBefore != engine.MasteredStep && c.StepIndex == engine.MasteredStep
	fb.TopLeft = 0
	if fb.Correct && c.StepIndex == engine.TopStep {
		fb.TopLeft = engine.MasterAfter - c.TopStreak
	}
	s.results[item.result].Mastered = fb.Mastered
}

// Unmaster zet een beheerst woord terug in de herhalingen: morgen weer, op 90 dagen, waar het
// zich opnieuw MasterAfter keer moet bewijzen. false als het concept niet bestaat of niet beheerst is.
func (s *Session) Unmaster(id string) (bool, error) {
	i := s.lib.Index(id)
	if i < 0 || s.lib.Concepts[i].StepIndex != engine.MasteredStep {
		return false, nil
	}
	c := &s.lib.Concepts[i]
	c.StepIndex, c.TopStreak, c.NextReview = engine.TopStep, 0, engine.ReviewDate(time.Now(), 1)
	return true, s.Save()
}

// formOf kijkt of een fout antwoord naar het Hongaars een andere vorm van het gevraagde woord is:
// olvasok of olvasni bij olvas, házban bij ház. Bij een infinitief (olvasni) wordt ook de 3e
// persoon (olvas) herkend, want daaruit maken de regels de andere vormen.
func (s *Session) formOf(c *storage.TranslationConcept, input string) *grammar.Match {
	if c.WordType == "" || len(c.TargetWordIDs) == 0 || s.lib.Words[c.TargetWordIDs[0]].Language != "hu" {
		return nil
	}
	for _, id := range c.TargetWordIDs {
		hu := strings.TrimSpace(s.lib.Words[id].Text)
		w := grammar.Word{Hungarian: hu, Type: c.WordType, Forms: c.Forms}
		if c.WordType == "ww" && strings.HasSuffix(hu, "ni") {
			third, ok := s.thirdPersonOf(hu)
			if !ok {
				continue
			}
			if engine.Fold(input) == engine.Fold(third.Hungarian) {
				return &grammar.Match{Word: third.Hungarian, Rule: "ww-onbep", RuleName: "3e persoon enkelvoud"}
			}
			w = third
		}
		if m, ok := grammar.Analyze(w, input); ok {
			return &m
		}
	}
	return nil
}

// thirdPersonOf zoekt in je lijst het werkwoord in de 3e persoon bij een infinitief (olvasni -> olvas)
func (s *Session) thirdPersonOf(infinitive string) (grammar.Word, bool) {
	for _, c := range s.lib.Concepts {
		if c.WordType != "ww" || len(c.TargetWordIDs) == 0 || s.lib.Words[c.SourceWordID].Language != "nl" {
			continue
		}
		for _, id := range c.TargetWordIDs {
			hu := strings.TrimSpace(s.lib.Words[id].Text)
			if strings.HasSuffix(hu, "ni") {
				continue
			}
			w := grammar.Word{Hungarian: hu, Type: "ww", Forms: c.Forms}
			inf := c.Forms["inf"]
			if inf == "" {
				if f, ok := grammar.Infinitive(hu); ok {
					inf = f.Text
				} else {
					inf = hu + "ni"
				}
			}
			if inf == infinitive {
				return w, true
			}
		}
	}
	return grammar.Word{}, false
}

// otherMeaning zoekt een ander concept met hetzelfde bronwoord (taal + tekst, niet het ID)
// waarvoor de invoer wél het goede antwoord is
func (s *Session) otherMeaning(current *storage.TranslationConcept, input string) *OtherMeaning {
	src := s.lib.Words[current.SourceWordID]
	for _, c := range s.lib.Concepts {
		other := s.lib.Words[c.SourceWordID]
		if c.ID == current.ID || other.Language != src.Language || !strings.EqualFold(strings.TrimSpace(other.Text), strings.TrimSpace(src.Text)) {
			continue
		}
		targets := s.targetTexts(&c)
		if ok, _ := engine.CheckAnswer(input, targets); ok {
			return &OtherMeaning{Answer: strings.Join(targets, ", "), Context: c.ContextSource}
		}
	}
	return nil
}
