package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"time"
)

// GrammarAttempt is één antwoord in de regeloefening (Regels), los van de herhalingen.
// Eén regel JSON per antwoord in regels.jsonl.
type GrammarAttempt struct {
	Time    time.Time `json:"time"`
	Rule    string    `json:"rule"`             // grammar.Rule.ID, bijv. "ban"
	Word    string    `json:"word"`             // het Hongaarse woord, bijv. "ház"
	Person  int       `json:"person,omitempty"` // bij werkwoorden (0 = én); anders 0
	Input   string    `json:"input"`
	Correct bool      `json:"correct"`
}

// AppendGrammar voegt een antwoord toe aan het regellog
func AppendGrammar(path string, a GrammarAttempt) error {
	return appendJSONLine(path, a)
}

// ReadGrammar leest het regellog. Een ontbrekend bestand is een lege geschiedenis; onleesbare
// regels worden overgeslagen.
func ReadGrammar(path string) ([]GrammarAttempt, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []GrammarAttempt
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var a GrammarAttempt
		if json.Unmarshal(sc.Bytes(), &a) == nil && a.Rule != "" {
			out = append(out, a)
		}
	}
	return out, sc.Err()
}
