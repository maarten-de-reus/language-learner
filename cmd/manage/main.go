package main

import (
	"bufio"
	"fmt"
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

	// Laad de bestaande data in
	wordsPool, conceptsDB, err := storage.LoadData(WordsPath, ProgressPath)
	if err != nil {
		fmt.Printf("⚠️ Kon %s niet laden, we starten een nieuwe database.\n", WordsPath)
		wordsPool = make(map[string]storage.Word)
		conceptsDB = []storage.TranslationConcept{}
	}

	fmt.Println("🛠️ Hungarian Language Learner - Beheeromgeving 🛠️")
	fmt.Println("--------------------------------------------------")

	for {
		fmt.Println("\nMaak een keuze:")
		fmt.Println("1) Voeg een nieuw Nederlands -> Hongaars concept toe")
		fmt.Println("2) Toon het totaal aantal woorden en concepten")
		fmt.Println("3) Stop en sla op")
		fmt.Print("Kies een optie (1-3): ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			// 1. Voer het Nederlandse woord in
			fmt.Print("\nVoer het Nederlandse woord in (NL): ")
			dutchText, _ := reader.ReadString('\n')
			dutchText = strings.TrimSpace(dutchText)
			if dutchText == "" {
				fmt.Println("❌ Woord mag niet leeg zijn.")
				continue
			}

			// 2. Voer de context in
			fmt.Print("Voer de context/hint in (bijv. 'Meubel' of 'Geld'): ")
			contextText, _ := reader.ReadString('\n')
			contextText = strings.TrimSpace(contextText)

			// 3. Voer de Hongaarse vertaling in
			fmt.Print("Voer de Hongaarse vertaling in (meerdere scheiden met een komma): ")
			hungarianInput, _ := reader.ReadString('\n')
			hungarianInput = strings.TrimSpace(hungarianInput)
			if hungarianInput == "" {
				fmt.Println("❌ Vertaling mag niet leeg zijn.")
				continue
			}

			// --- SLIMME ID GENERATIE ---
			safeWordNL := strings.ReplaceAll(strings.ToLower(dutchText), " ", "_")
			safeContext := strings.ReplaceAll(strings.ToLower(contextText), " ", "_")
			
			// We maken twee verschillende Concept-IDs aan voor de twee leerrichtingen
			conceptID_NL_HU := "c_nl_hu_" + safeWordNL
			if safeContext != "" {
				conceptID_NL_HU = "c_nl_hu_" + safeWordNL + "_" + safeContext
			}
			
			// --- DUPLICATEN CHECK ---
			// We controleren of de NL->HU variant al bestaat (als die er is, is de HU->NL er vaak ook)
			duplicateFound := false
			for _, existingConcept := range conceptsDB {
				if existingConcept.ID == conceptID_NL_HU {
					duplicateFound = true
					break
				}
			}

			if duplicateFound {
				fmt.Printf("⚠️  Fout: Dit woord met deze context bestaat al in de database!\n")
				continue
			}

			// --- VERWERK DE WOORDEN IN DE POOL ---
			// 1. Voeg het Nederlandse woord toe
			cleanDutchID := "w_nl_" + safeWordNL
			wordsPool[cleanDutchID] = storage.Word{
				ID:       cleanDutchID,
				Language: "nl",
				Text:     dutchText,
			}

			// 2. Voeg alle Hongaarse synoniemen toe
			hungarianWords := strings.Split(hungarianInput, ",")
			var targetIDs_HU []string

			for _, huWord := range hungarianWords {
				huWord = strings.TrimSpace(huWord)
				if huWord == "" {
					continue
				}
				cleanHuID := "w_hu_" + strings.ReplaceAll(strings.ToLower(huWord), " ", "_")
				
				wordsPool[cleanHuID] = storage.Word{
					ID:       cleanHuID,
					Language: "hu",
					Text:     huWord,
				}
				targetIDs_HU = append(targetIDs_HU, cleanHuID)
			}

			// --- CONCEPT 1: Nederlands -> Hongaars ---
			newConceptNLtoHU := storage.TranslationConcept{
				ID:            conceptID_NL_HU,
				SourceWordID:  cleanDutchID,
				ContextSource: contextText, // Context in het Nederlands
				TargetWordIDs: targetIDs_HU,
			}
			conceptsDB = append(conceptsDB, newConceptNLtoHU)

			// --- CONCEPT 2: Hongaars -> Nederlands (AUTOMATISCH) ---
			// We pakken het eerste Hongaarse woord uit de lijst als 'Bron' voor de omgekeerde richting
			firstHuWordClean := strings.ReplaceAll(strings.ToLower(hungarianWords[0]), " ", "_")
			conceptID_HU_NL := "c_hu_nl_" + firstHuWordClean
			if safeContext != "" {
				conceptID_HU_NL = "c_hu_nl_" + firstHuWordClean + "_" + safeContext
			}

			newConceptHUtoNL := storage.TranslationConcept{
				ID:            conceptID_HU_NL,
				SourceWordID:  targetIDs_HU[0], // Het Hongaarse woord is nu de bron!
				ContextSource: "",              // Voor een Hongaar is de NL context vaak overbodig, of laat hem leeg
				TargetWordIDs: []string{cleanDutchID}, // Nederlands is nu het doel
			}
			conceptsDB = append(conceptsDB, newConceptHUtoNL)

			fmt.Printf("🎉 Dubbelslag! Twee concepten aangemaakt:\n")
			fmt.Printf("   👉 🇳🇱 ➔ 🇭🇺 : '%s' (%s) ➔ %s\n", dutchText, contextText, strings.Join(hungarianWords, ", "))
			fmt.Printf("   👉 🇭🇺 ➔ 🇳🇱 : '%s' ➔ '%s'\n", hungarianWords[0], dutchText)

		case "2":
			fmt.Printf("\n📊 Statistieken:\n")
			fmt.Printf("   - Totaal aantal unieke woorden in de pool: %d\n", len(wordsPool))
			fmt.Printf("   - Totaal aantal actieve leerconcepten: %d\n", len(conceptsDB))

		case "3":
			// Sla de data definitief op
			err := storage.SaveWords(WordsPath, wordsPool, conceptsDB)
			if err != nil {
				fmt.Printf("❌ Fout bij het opslaan van woorden: %v\n", err)
			} else {
				fmt.Println("💾 Wijzigingen succesvol opgeslagen in woorden.json!")
			}
			fmt.Println("Tot ziens! Viszlát!")
			return

		default:
			fmt.Println("❌ Ongeldige keuze, probeer opnieuw.")
		}
	}
}
