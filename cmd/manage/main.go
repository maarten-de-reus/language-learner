package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"hungarian-language-learner/internal/storage"
)

const (
	WordsPath    = "woorden.json"
	ProgressPath = "voortgang.json"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Laad de bestaande data in. Alleen als woorden.json echt nog niet bestaat beginnen we leeg;
	// bij elke andere fout stoppen we, anders overschrijft "opslaan" de bestaande woordenlijst.
	wordsPool, conceptsDB, err := storage.LoadData(WordsPath, ProgressPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fmt.Printf("ℹ️  %s bestaat nog niet, we starten een nieuwe database.\n", WordsPath)
		wordsPool = make(map[string]storage.Word)
		conceptsDB = []storage.TranslationConcept{}
	case err != nil:
		fmt.Printf("❌ Kon de data niet laden: %v\n", err)
		fmt.Println("   Herstel het bestand (bijv. met git) en probeer opnieuw.")
		os.Exit(1)
	}

	fmt.Println("🛠️ Hungarian Language Learner - Beheeromgeving 🛠️")
	fmt.Println("--------------------------------------------------")

	unsaved := 0
	for {
		fmt.Println("\nMaak een keuze:")
		fmt.Println("1) Voeg een nieuw Nederlands -> Hongaars concept toe")
		fmt.Println("2) Toon het totaal aantal woorden en concepten")
		fmt.Println("3) Stop en sla op")
		fmt.Print("Kies een optie (1-3): ")

		choice, eof := readLine(reader)
		if eof {
			fmt.Printf("\n⚠️  Invoer gesloten, gestopt zonder op te slaan (%d niet-opgeslagen concepten).\n", unsaved)
			return
		}

		switch choice {
		case "1":
			fmt.Print("\nVoer het Nederlandse woord in (NL): ")
			dutchText, _ := readLine(reader)

			fmt.Print("Voer de context/hint in (bijv. 'Meubel' of 'Geld'): ")
			contextText, _ := readLine(reader)

			fmt.Print("Voer de Hongaarse vertaling in (meerdere scheiden met een komma): ")
			hungarianInput, _ := readLine(reader)

			var res addResult
			conceptsDB, res, err = addConceptPair(wordsPool, conceptsDB, dutchText, contextText, hungarianInput)
			if err != nil {
				fmt.Printf("❌ %s.\n", strings.ToUpper(err.Error()[:1])+err.Error()[1:])
				continue
			}
			unsaved++

			fmt.Printf("🎉 Toegevoegd:\n")
			fmt.Printf("   👉 🇳🇱 ➔ 🇭🇺 : '%s' (%s) ➔ %s\n", dutchText, contextText, strings.Join(res.Hungarian, ", "))
			if res.ReverseMerged {
				fmt.Printf("   👉 🇭🇺 ➔ 🇳🇱 : '%s' bestond al, '%s' is toegevoegd als extra goed antwoord\n", res.ReverseSource, dutchText)
			} else {
				fmt.Printf("   👉 🇭🇺 ➔ 🇳🇱 : '%s' ➔ '%s'\n", res.ReverseSource, dutchText)
			}

		case "2":
			fmt.Printf("\n📊 Statistieken:\n")
			fmt.Printf("   - Totaal aantal unieke woorden in de pool: %d\n", len(wordsPool))
			fmt.Printf("   - Totaal aantal actieve leerconcepten: %d\n", len(conceptsDB))

		case "3":
			if err := storage.SaveWords(WordsPath, wordsPool, conceptsDB); err != nil {
				// Niet afsluiten: de nieuwe woorden staan alleen nog in het geheugen
				fmt.Printf("❌ Fout bij het opslaan van woorden: %v\n", err)
				fmt.Println("   Je wijzigingen zijn niet verloren; los het probleem op en kies opnieuw 3.")
				continue
			}
			fmt.Println("💾 Wijzigingen succesvol opgeslagen in woorden.json!")
			fmt.Println("Tot ziens! Viszlát!")
			return

		default:
			fmt.Println("❌ Ongeldige keuze, probeer opnieuw.")
		}
	}
}

// readLine leest één regel zonder witruimte; eof is true als de invoer gesloten is (Ctrl+D)
func readLine(r *bufio.Reader) (line string, eof bool) {
	s, err := r.ReadString('\n')
	return strings.TrimSpace(s), errors.Is(err, io.EOF) && s == ""
}
