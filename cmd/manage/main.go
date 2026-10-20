package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"hungarian-language-learner/internal/catalog"
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
	lib, err := storage.LoadLibrary(WordsPath, ProgressPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fmt.Printf("ℹ️  %s bestaat nog niet, we starten een nieuwe database.\n", WordsPath)
		lib = &storage.Library{Words: map[string]storage.Word{}}
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

			nc, err := catalog.ParseInput(dutchText, contextText, hungarianInput)
			if err != nil {
				printError(err)
				continue
			}
			if catalog.IsDuplicate(lib, nc) {
				printError(catalog.ErrDuplicate)
				continue
			}
			askReverse(reader, lib, &nc)

			fmt.Printf("Voorbeeldzin in het Hongaars met '%s' erin (optioneel, Enter = overslaan): ", nc.Hungarian[0])
			nc.Example, _ = readLine(reader)

			res, err := catalog.AddPair(lib, nc)
			if err != nil {
				printError(err)
				continue
			}
			unsaved++

			fmt.Printf("🎉 Toegevoegd:\n")
			fmt.Printf("   👉 🇳🇱 ➔ 🇭🇺 : '%s'%s ➔ %s\n", nc.Dutch, hintSuffix(nc.Context), strings.Join(nc.Hungarian, ", "))
			if nc.Example != "" {
				fmt.Printf("   💬 %s\n", nc.Example)
			}
			if res.ReverseMerged {
				fmt.Printf("   👉 🇭🇺 ➔ 🇳🇱 : '%s' bestond al, '%s' is toegevoegd als extra goed antwoord\n", res.ReverseSource, nc.Dutch)
			} else {
				fmt.Printf("   👉 🇭🇺 ➔ 🇳🇱 : '%s'%s ➔ '%s'\n", res.ReverseSource, hintSuffix(nc.ReverseContext), nc.Dutch)
			}

		case "2":
			fmt.Printf("\n📊 Statistieken:\n")
			fmt.Printf("   - Totaal aantal unieke woorden in de pool: %d\n", len(lib.Words))
			fmt.Printf("   - Totaal aantal actieve leerconcepten: %d\n", len(lib.Concepts))

		case "3":
			if err := storage.SaveWords(WordsPath, lib.Words, lib.Concepts); err != nil {
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

// askReverse bepaalt hoe het HU->NL concept eruitziet. Bestaat het Hongaarse woord al als HU->NL,
// dan vragen we of het dezelfde betekenis is (extra goed antwoord, zoals ő -> hij/zij)
// of een andere (apart concept met eigen hint, zoals idő -> weer / tijd).
func askReverse(reader *bufio.Reader, lib *storage.Library, nc *catalog.NewPair) {
	hu := nc.Hungarian[0]
	candidates := catalog.ReverseCandidates(lib, hu)

	if len(candidates) > 0 {
		fmt.Printf("\n'%s' bestaat al als HU->NL. Is '%s' dezelfde betekenis of een andere?\n", hu, nc.Dutch)
		for n, i := range candidates {
			c := lib.Concepts[i]
			fmt.Printf("  %d) Zelfde betekenis als '%s'%s ➔ %s (extra goed antwoord)\n", n+1, hu, hintSuffix(c.ContextSource), answers(lib, c))
		}
		newOption := len(candidates) + 1
		fmt.Printf("  %d) Andere betekenis (apart concept met eigen hint)\n", newOption)

		for {
			fmt.Printf("Kies (1-%d): ", newOption)
			line, eof := readLine(reader)
			var n int
			if _, err := fmt.Sscan(line, &n); err == nil && n >= 1 && n < newOption {
				nc.MergeReverseInto = lib.Concepts[candidates[n-1]].ID
				return
			}
			if n == newOption || eof {
				break
			}
		}

		// Bestaande concepten zonder hint zijn na deze toevoeging niet meer te onderscheiden
		nc.ExistingHints = map[string]string{}
		for _, i := range candidates {
			c := lib.Concepts[i]
			if strings.TrimSpace(c.ContextSource) != "" {
				continue
			}
			for {
				fmt.Printf("Het bestaande '%s' ➔ %s heeft nog geen hint. Hint (bijv. 'buiten, ☀️'): ", hu, answers(lib, c))
				hint, eof := readLine(reader)
				if hint != "" || eof {
					nc.ExistingHints[c.ID] = hint
					break
				}
			}
		}
	}

	for {
		fmt.Printf("Hint voor HU->NL, zodat duidelijk is welke betekenis bedoeld wordt [Enter = '%s']: ", nc.Context)
		hint, eof := readLine(reader)
		if hint == "" {
			hint = nc.Context
		}
		if eof || len(candidates) == 0 || (hint != "" && !catalog.ReverseHintTaken(lib, *nc, hint)) {
			nc.ReverseContext = hint
			return
		}
		fmt.Println("❌ Die hint onderscheidt deze betekenis niet van de bestaande; kies een andere.")
	}
}

func answers(lib *storage.Library, c storage.TranslationConcept) string {
	return strings.Join(catalog.TargetTexts(lib, c), ", ")
}

func hintSuffix(hint string) string {
	if hint == "" {
		return ""
	}
	return " (" + hint + ")"
}

func printError(err error) {
	msg := err.Error()
	fmt.Printf("❌ %s.\n", strings.ToUpper(msg[:1])+msg[1:])
}

// readLine leest één regel zonder witruimte; eof is true als de invoer gesloten is (Ctrl+D)
func readLine(r *bufio.Reader) (line string, eof bool) {
	s, err := r.ReadString('\n')
	return strings.TrimSpace(s), errors.Is(err, io.EOF) && s == ""
}
