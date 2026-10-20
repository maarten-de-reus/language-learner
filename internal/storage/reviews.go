package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"time"
)

// Review is één gegeven antwoord. Samen vormen ze de geschiedenis waaruit statistieken
// (moeilijkste woorden, hoe vaak je oefent) af te leiden zijn. Eén regel JSON per antwoord,
// zodat het bestand alleen groeit en later één-op-één naar een databasetabel kan.
type Review struct {
	Time         time.Time `json:"time"`
	ConceptID    string    `json:"concept_id"`
	Input        string    `json:"input"`
	Correct      bool      `json:"correct"`
	AccentOnly   bool      `json:"accent_only,omitempty"`   // fout, maar alleen in de accenten
	OtherMeaning bool      `json:"other_meaning,omitempty"` // fout, maar wel een andere betekenis van het woord
	Retry        bool      `json:"retry,omitempty"`         // oefenherhaling in dezelfde run; telt niet mee voor de planning
	StepBefore   int       `json:"step_before"`
	StepAfter    int       `json:"step_after"`
	NextReview   time.Time `json:"next_review"`
}

// ReviewLog schrijft antwoorden achteraan een JSON Lines-bestand
type ReviewLog struct {
	path string
}

func NewReviewLog(path string) *ReviewLog {
	return &ReviewLog{path: path}
}

// Append voegt één antwoord toe. Bestaande regels worden nooit aangeraakt.
func (l *ReviewLog) Append(r Review) error {
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	// Eindigt het bestand niet op een newline (half geschreven regel na een crash),
	// begin dan op een nieuwe regel, zodat dit antwoord niet aan de kapotte regel vastplakt
	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		last := make([]byte, 1)
		if _, err := f.ReadAt(last, info.Size()-1); err == nil && last[0] != '\n' {
			line = append([]byte{'\n'}, line...)
		}
	}
	// Eén Write per regel: met O_APPEND komt die in zijn geheel achteraan
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// ReadReviews leest alle antwoorden. Een ontbrekend bestand is een lege geschiedenis.
// Onleesbare regels (bijv. een half geschreven laatste regel na een crash) worden overgeslagen
// en geteld, zodat één kapotte regel de rest van de geschiedenis niet onbruikbaar maakt.
func ReadReviews(path string) (reviews []Review, skipped int, err error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var r Review
		if json.Unmarshal(scanner.Bytes(), &r) != nil {
			skipped++
			continue
		}
		reviews = append(reviews, r)
	}
	return reviews, skipped, scanner.Err()
}
