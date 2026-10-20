package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/quiz"
	"hungarian-language-learner/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

// De bestandspaden als globale constanten
const (
	WordsPath    = "woorden.json"
	ProgressPath = "voortgang.json"
	RunSize      = 10
)

type model struct {
	session   *quiz.Session
	userInput string
	saveError string
}

func initialModel() model {
	lib, err := storage.LoadLibrary(WordsPath, ProgressPath)
	if err != nil {
		fmt.Printf("Fout bij het laden van data: %v\n", err)
		os.Exit(1)
	}
	return model{session: quiz.New(lib, ProgressPath, RunSize)}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	answering := !m.session.Done() && m.session.Feedback() == nil

	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		// Voortgang wordt na elk antwoord al opgeslagen
		return m, tea.Quit

	case tea.KeyEnter:
		switch {
		case m.session.Done():
			m.session.NewRun()
		case m.session.Feedback() != nil:
			m.session.Next()
		default:
			if _, err := m.session.Answer(m.userInput); err != nil {
				m.saveError = err.Error()
			} else {
				m.saveError = ""
			}
			return m, nil
		}
		m.userInput = ""

	case tea.KeyBackspace:
		if answering {
			runes := []rune(m.userInput)
			if len(runes) > 0 {
				m.userInput = string(runes[:len(runes)-1])
			}
		}

	case tea.KeySpace:
		if answering {
			m.userInput += " "
		}

	default:
		// Vang multibyte karakters (Hongaarse accenten) correct op via msg.Runes
		if answering && len(key.Runes) > 0 {
			m.userInput += string(key.Runes)
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("🚀 Gelaagde Hongaarse Woorden Trainer 🚀\n")
	b.WriteString("--------------------------------------------------\n\n")

	if m.saveError != "" {
		fmt.Fprintf(&b, "⚠️  Voortgang kon niet worden opgeslagen: %s\n\n", m.saveError)
	}

	stats := m.session.Stats()
	if m.session.Done() {
		results := m.session.Results()
		if len(results) == 0 {
			b.WriteString("✅ Er staan op dit moment geen woorden klaar om te herhalen.\n")
		} else {
			good := 0
			for _, r := range results {
				if r.Correct {
					good++
				}
			}
			fmt.Fprintf(&b, "🎉 Run voltooid! Score: %d / %d\n", good, len(results))
			for _, r := range results {
				if r.Retries == 0 {
					continue
				}
				if r.RetryCorrect {
					fmt.Fprintf(&b, "   ↻ %s: fout, goed bij herhaling %d\n", r.Prompt, r.Retries)
				} else {
					fmt.Fprintf(&b, "   ↻ %s: ook bij %d herhaling(en) fout (%s)\n", r.Prompt, r.Retries, strings.Join(r.Expected, ", "))
				}
			}
			fmt.Fprintf(&b, "👉 Er staan nog %d woorden te wachten in de database.\n", stats.Due)
		}
		if stats.Due == 0 && stats.NextDue != nil {
			fmt.Fprintf(&b, "⏰ Volgende herhaling: %s\n", relative(*stats.NextDue))
		}
		b.WriteString("\n[Druk op ENTER voor een nieuwe run • Esc of Ctrl+C om te stoppen]\n")
		return b.String()
	}

	card := m.session.Current()
	interval := fmt.Sprintf("Interval: %d dagen", engine.AnkiIntervals[card.Step])
	if engine.AnkiIntervals[card.Step] == 1 {
		interval = "Interval: 1 dag"
	}
	if card.Retry {
		interval = "↻ herhaling, telt niet mee voor de planning"
	}
	fmt.Fprintf(&b, "Vraag %d van de %d uit deze run [%s]\n", card.Position, card.RunSize, interval)
	if card.Context != "" {
		fmt.Fprintf(&b, "Context: %s\n", card.Context)
	}
	fmt.Fprintf(&b, "Vertaal het woord: '%s'\n", card.Prompt)
	if card.Example != "" {
		fmt.Fprintf(&b, "💬 \"%s\"\n", card.Example)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "Jouw antwoord: %s", m.userInput)

	fb := m.session.Feedback()
	if fb == nil {
		b.WriteString("_\n\n[Typ je antwoord • Enter om te controleren]\n")
		return b.String()
	}

	answers := strings.Join(fb.Expected, ", ")
	if fb.Correct {
		fmt.Fprintf(&b, "\n\n✅ Correct! Opties: %s.\n", answers)
	} else {
		fmt.Fprintf(&b, "\n\n❌ Fout! Het moest zijn: %s.\n", answers)
		if om := fb.OtherMeaning; om != nil {
			ctx := ""
			if om.Context != "" {
				ctx = " (" + om.Context + ")"
			}
			fmt.Fprintf(&b, "💡 Klopt, '%s' kan ook '%s'%s betekenen, maar hier wordt '%s' bedoeld.\n", card.Prompt, om.Answer, ctx, answers)
		}
		if fb.AccentOnly {
			b.WriteString("💡 Bijna! Alleen de accenten kloppen niet.\n")
		}
	}
	if fb.Example != "" && card.Example == "" {
		fmt.Fprintf(&b, "💬 \"%s\"\n", fb.Example)
	}
	fmt.Fprintf(&b, "Volgende herhaling: %s\n", relative(fb.NextReview))
	if fb.WillRetry {
		b.WriteString("↻ Je krijgt dit woord aan het eind van de ronde nog een keer.\n")
	}
	b.WriteString("\n")
	b.WriteString("[Druk op ENTER voor het volgende woord]\n")
	return b.String()
}

// relative geeft een tijdstip weer als "morgen", "over 7 dagen" of "over 3 uur"
func relative(t time.Time) string {
	d := time.Until(t)
	switch {
	case d < time.Hour:
		return "binnen een uur"
	case d < 20*time.Hour:
		return fmt.Sprintf("over %d uur", int(d.Hours()+0.5))
	case d < 36*time.Hour:
		return "morgen"
	default:
		return fmt.Sprintf("over %d dagen", int(d.Hours()/24+0.5))
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fout: %v", err)
		os.Exit(1)
	}
}
