package main

import (
	"fmt"
	"os"
	"strings"

	"hungarian-language-learner/internal/engine"
	"hungarian-language-learner/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

// De bestandspaden als globale constanten
const (
	WordsPath    = "woorden.json"
	ProgressPath = "voortgang.json"
)

type sessionState int

const (
	stateQuiz sessionState = iota
	stateRunComplete
)

// HIER ZIT DE FIX: De juiste types voor de gelaagde architectuur
type model struct {
	words        map[string]storage.Word              // Woorden-pool om ID's te vertalen
	concepts     []storage.TranslationConcept          // De volledige lijst uit JSON
	currentRun   []storage.TranslationConcept          // De huidige actieve 'run' chunk
	currentIndex int
	state        sessionState
	userInput    string
	feedback     string
	showAnswer   bool
	remaining    int
}

func initialModel() model {
	// Laad de data in vanuit de internal storage package
	wordsPool, conceptsDB, err := storage.LoadData(WordsPath, ProgressPath)
	if err != nil {
		fmt.Printf("Fout bij het laden van data: %v\n", err)
		os.Exit(1)
	}

	// Vraag de eerste run op aan de engine (bijv. max 10 woorden)
	run, remaining := engine.GetNextChunk(conceptsDB, 10)

	state := stateQuiz
	if len(run) == 0 {
		state = stateRunComplete
	}

	return model{
		words:        wordsPool,
		concepts:     conceptsDB,
		currentRun:   run,
		currentIndex: 0,
		state:        state,
		userInput:    "",
		feedback:     "",
		showAnswer:   false,
		remaining:    remaining,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Als de huidige run leeg is (of alle woorden zijn gedaan), zet de status op compleet
	if len(m.currentRun) == 0 {
		m.state = stateRunComplete
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Automatisch opslaan bij afsluiten
			_ = storage.SaveProgress(ProgressPath, m.concepts)
			return m, tea.Quit

		case tea.KeyEnter:
			if m.state == stateRunComplete {
				// Sla voortgang op en start een nieuwe run
				_ = storage.SaveProgress(ProgressPath, m.concepts)
				
				m.currentRun, m.remaining = engine.GetNextChunk(m.concepts, 10)
				m.currentIndex = 0
				m.state = stateQuiz
				if len(m.currentRun) == 0 {
					m.state = stateRunComplete
				}
				m.feedback = ""
				return m, nil
			}

			if m.showAnswer {
				m.currentIndex++
				if m.currentIndex >= len(m.currentRun) {
					m.state = stateRunComplete
					_ = storage.SaveProgress(ProgressPath, m.concepts)
				}
				m.userInput = ""
				m.feedback = ""
				m.showAnswer = false
				return m, nil
			}

			// Controleer het antwoord
			currentConcept := &m.currentRun[m.currentIndex]
			cleanedInput := strings.TrimSpace(strings.ToLower(m.userInput))
			correct := false

			var correctTexts []string
			for _, targetID := range currentConcept.TargetWordIDs {
				targetWord := m.words[targetID]
				correctTexts = append(correctTexts, targetWord.Text)
				if cleanedInput == strings.ToLower(targetWord.Text) {
					correct = true
				}
			}

			// Bereken de nieuwe Spaced Repetition status via de engine
			engine.UpdateProgressSmart(currentConcept, correct)

			// Update de hoofdlijst zodat de voortgang straks correct wordt opgeslagen
			for i, c := range m.concepts {
				if c.ID == currentConcept.ID {
					m.concepts[i] = *currentConcept
				}
			}

			if correct {
				m.feedback = fmt.Sprintf("✅ Correct! Opties: %s. Volgende stap: %d", strings.Join(correctTexts, ", "), currentConcept.StepIndex)
			} else {
				m.feedback = fmt.Sprintf("❌ Fout! Het moest zijn: %s. Reset naar stap 0.", strings.Join(correctTexts, ", "))
			}
			m.showAnswer = true
			return m, nil

		case tea.KeyBackspace:
			if !m.showAnswer {
				runes := []rune(m.userInput)
				if len(runes) > 0 {
					m.userInput = string(runes[:len(runes)-1])
				}
			}
			return m, nil

		case tea.KeySpace:
			if !m.showAnswer {
				m.userInput += " "
			}
			return m, nil
		}

		// Vang multibyte karakters (Hongaarse accenten) correct op via msg.Runes
		if !m.showAnswer && len(msg.Runes) > 0 {
			m.userInput += string(msg.Runes)
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "🚀 Gelaagde Hongaarse Woorden Trainer 🚀\n"
	s += "--------------------------------------------------\n\n"

	if m.state == stateRunComplete && len(m.currentRun) == 0 {
		s += "✅ Er staan op dit moment geen woorden klaar om te herhalen.\n\n"
		s += "[Druk op ENTER om opnieuw te controleren • Esc of Ctrl+C om te stoppen]\n"
		return s
	}

	if m.state == stateRunComplete {
		s += "🎉 Run voltooid!\n"
		s += fmt.Sprintf("👉 Er staan nog %d woorden te wachten in de database.\n\n", m.remaining)
		s += "[Druk op ENTER om een nieuwe run te starten • Esc of Ctrl+C om te stoppen]\n"
		return s
	}

	currentConcept := m.currentRun[m.currentIndex]
	sourceWord := m.words[currentConcept.SourceWordID]

	s += fmt.Sprintf("Vraag %d van de %d uit deze run [Huidige stap: %d]\n", m.currentIndex+1, len(m.currentRun), currentConcept.StepIndex)
	s += fmt.Sprintf("Context: %s\n", currentConcept.ContextSource)
	s += fmt.Sprintf("Vertaal het woord: '%s'\n\n", sourceWord.Text)
	s += fmt.Sprintf("Jouw antwoord: %s", m.userInput)
	if !m.showAnswer {
		s += "_"
	}
	s += "\n\n"

	if m.feedback != "" {
		s += m.feedback + "\n\n"
	}

	if m.showAnswer {
		s += "[Druk op ENTER voor het volgende woord]\n"
	} else {
		s += "[Typ je antwoord • Enter om te controleren]\n"
	}

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fout: %v", err)
		os.Exit(1)
	}
}
