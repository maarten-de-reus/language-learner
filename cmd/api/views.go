package main

import (
	"encoding/json"
	"net/http"

	"hungarian-language-learner/internal/quiz"
)

// stateView is het antwoord op elk /api-request; de frontend rendert volledig op basis hiervan
type stateView struct {
	Card      *quiz.Card     `json:"card"`
	Feedback  *quiz.Feedback `json:"feedback"`
	Done      bool           `json:"done"`
	Results   []quiz.Result  `json:"results"`
	Stats     quiz.Stats     `json:"stats"`
	RulesDue  int            `json:"rulesDue"` // grammaticaregels aan de beurt (Oefenen -> Regels)
	SaveError string         `json:"saveError,omitempty"`
}

func (s *server) writeState(w http.ResponseWriter) {
	v := stateView{
		Card:      s.session.Current(),
		Feedback:  s.session.Feedback(),
		Done:      s.session.Done(),
		Results:   s.session.Results(),
		Stats:     s.session.Stats(),
		SaveError: s.saveError,
		RulesDue:  s.rulesDue(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}
