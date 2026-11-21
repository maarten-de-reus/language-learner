// Regels: grammatica oefenen met je eigen woorden, los van de herhalingen.

package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"slices"
	"time"

	"hungarian-language-learner/internal/grammar"
	"hungarian-language-learner/internal/storage"
)

type grammarRuleView struct {
	grammar.Rule
	Active   bool              `json:"active"`
	Score    grammar.Score     `json:"score"`
	Words    int               `json:"words"` // zoveel van je woorden kunnen met deze regel
	Examples []grammar.Example `json:"examples"`
}

type grammarView struct {
	Rules   []grammarRuleView `json:"rules"`
	Active  []string          `json:"active"`
	Typed   int               `json:"typed"`   // woorden met een woordsoort
	Untyped int               `json:"untyped"` // NL->HU kaarten zonder woordsoort
}

// activeRules: je eigen keuze, of de standaardkeuze zolang je niets gekozen hebt
func (s *server) activeRules() []string {
	if s.settings.GrammarRules == nil {
		return grammar.DefaultActive
	}
	return s.settings.GrammarRules
}

func (s *server) grammarView() grammarView {
	words := grammar.Words(s.lib)
	scores := grammar.Scores(s.grammarLog)
	active := s.activeRules()
	v := grammarView{Active: active, Typed: len(words)}
	for _, c := range s.lib.Concepts {
		if c.WordType == "" && len(c.TargetWordIDs) > 0 && s.lib.Words[c.SourceWordID].Language == "nl" {
			v.Untyped++
		}
	}
	for _, r := range grammar.Rules {
		v.Rules = append(v.Rules, grammarRuleView{
			Rule: r, Active: slices.Contains(active, r.ID), Score: scores[r.ID],
			Words: len(r.Applicable(words)), Examples: r.Examples(words, 4),
		})
	}
	return v
}

func (s *server) handleGrammar(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, s.grammarView())
}

func (s *server) handleGrammarActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Active []string `json:"active"`
	}
	if !decode(w, r, &req) {
		return
	}
	active := []string{} // leeg = bewust alles uit (nil zou de standaardkeuze betekenen)
	for _, id := range req.Active {
		if _, ok := grammar.RuleByID(id); !ok {
			writeError(w, http.StatusBadRequest, fmt.Errorf("onbekende regel %q", id))
			return
		}
		if !slices.Contains(active, id) {
			active = append(active, id)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.settings
	next.GrammarRules = active
	if s.settingsPath != "" {
		if err := storage.SaveSettings(s.settingsPath, next); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	s.settings = next
	writeJSON(w, http.StatusOK, s.grammarView())
}

// handleGrammarExercise geeft een opgave. ?rule= beperkt tot één regel (ook als die niet actief
// is: een regel uitproberen vanuit de uitleg), ?avoid= is het woord van de vorige opgave.
func (s *server) handleGrammarExercise(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	active := s.activeRules()
	if id := r.URL.Query().Get("rule"); id != "" {
		active = []string{id}
	}
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	ex, ok := grammar.Next(grammar.Words(s.lib), active, grammar.Scores(s.grammarLog), r.URL.Query().Get("avoid"), rng)
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("geen opgave: zet een regel aan, of geef je woorden een woordsoort (Beheer)"))
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

type grammarCheckRequest struct {
	Rule   string `json:"rule"`
	Word   string `json:"word"`
	Person int    `json:"person"`
	Input  string `json:"input"`
}

type grammarCheckResponse struct {
	grammar.Result
	Score     grammar.Score `json:"score"`               // de score van deze regel, na dit antwoord
	SaveError string        `json:"saveError,omitempty"` // het antwoord kon niet in het regellog
}

func (s *server) handleGrammarCheck(w http.ResponseWriter, r *http.Request) {
	var req grammarCheckRequest
	if !decode(w, r, &req) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := grammar.Check(req.Rule, grammar.FindWord(grammar.Words(s.lib), req.Word), req.Person, req.Input)
	if !ok {
		writeError(w, http.StatusBadRequest, errors.New("deze opgave bestaat niet (meer)"))
		return
	}
	a := storage.GrammarAttempt{Time: time.Now(), Rule: req.Rule, Word: req.Word, Person: req.Person, Input: req.Input, Correct: res.Correct}
	resp := grammarCheckResponse{Result: res}
	if s.grammarPath != "" {
		if err := storage.AppendGrammar(s.grammarPath, a); err != nil {
			log.Printf("Regellog niet bijgewerkt: %v", err)
			resp.SaveError = "je antwoord is niet opgeslagen: " + err.Error()
		}
	}
	if resp.SaveError == "" {
		s.grammarLog = append(s.grammarLog, a)
	}
	resp.Score = grammar.Scores(s.grammarLog)[req.Rule]
	writeJSON(w, http.StatusOK, resp)
}
