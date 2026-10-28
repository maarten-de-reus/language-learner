package catalog

import (
	"errors"
	"fmt"
	"strings"

	"hungarian-language-learner/internal/storage"
)

// ImportLine is één regel uit een geplakte lijst: hongaars;nederlands;context;voorbeeldzin
type ImportLine struct {
	No        int    `json:"no"` // regelnummer in de geplakte tekst (vanaf 1)
	Hungarian string `json:"hungarian"`
	Dutch     string `json:"dutch"`
	Context   string `json:"context"`
	Example   string `json:"example"`
	Err       string `json:"error,omitempty"` // fout bij het lezen van de regel
}

// ParseImport leest een geplakte lijst. Kolommen worden gescheiden door ; (spaties eromheen
// mogen, hoeven niet). Context en voorbeeldzin zijn optioneel. Lege regels, regels die met #
// beginnen en een kopregel ("hongaars;nederlands;...") worden overgeslagen.
func ParseImport(text string) []ImportLine {
	var lines []ImportLine
	for i, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") || isHeader(raw) {
			continue
		}
		fields := strings.Split(raw, ";")
		for j := range fields {
			fields[j] = strings.TrimSpace(fields[j])
		}
		line := ImportLine{No: i + 1}
		switch {
		case len(fields) < 2:
			line.Err = "te weinig kolommen: minstens hongaars;nederlands"
		case len(fields) > 4:
			line.Err = "te veel kolommen: maximaal hongaars;nederlands;context;voorbeeldzin (een ; in de voorbeeldzin kan niet)"
		}
		if line.Err == "" {
			fields = append(fields, "", "")
			line.Hungarian, line.Dutch, line.Context, line.Example = fields[0], fields[1], fields[2], fields[3]
			switch {
			case line.Hungarian == "":
				line.Err = "Hongaars woord ontbreekt"
			case line.Dutch == "":
				line.Err = "Nederlands woord ontbreekt"
			case strings.Contains(line.Dutch, ","):
				line.Err = "meerdere Nederlandse woorden: zet elk op een eigen regel. Betekenen ze iets anders, " +
					"geef dan elke regel een eigen hint (nem;nee;antwoord en nem;niet;ontkenning)"
			}
		} else {
			line.Hungarian = fields[0]
		}
		lines = append(lines, line)
	}
	return lines
}

func isHeader(line string) bool {
	l := strings.ToLower(strings.ReplaceAll(line, " ", ""))
	return strings.HasPrefix(l, "hongaars;nederlands")
}

// ImportChoice zijn de keuzes voor een regel waarvan het Nederlandse en/of Hongaarse woord al bestaat
type ImportChoice struct {
	Forward *ForwardChoice `json:"forward,omitempty"` // het Nederlandse woord bestaat al; nil = nog niet gekozen
	Context *string        `json:"context,omitempty"` // hint die de context-kolom van deze regel vervangt (in het voorbeeld ingevuld)

	MergeInto     string            `json:"mergeInto"`     // bestaand HU->NL concept met dezelfde betekenis; leeg = andere betekenis
	ExistingHints map[string]string `json:"existingHints"` // bij een andere betekenis: hints voor bestaande concepten zonder hint
	Reverse       bool              `json:"reverse"`       // de keuze voor het Hongaarse woord is gemaakt
}

// ForwardChoice: Hongaarse woorden toevoegen aan een bestaande betekenis, of een nieuwe betekenis
type ForwardChoice struct {
	Extend string            `json:"extend"` // bestaand NL->HU concept om aan te vullen; leeg = nieuwe betekenis
	Hints  map[string]string `json:"hints"`  // bij een nieuwe betekenis: hints voor bestaande concepten zonder hint
}

// Line-statussen
const (
	StatusAdded     = "added"     // toegevoegd (of: wordt toegevoegd, bij controleren)
	StatusDuplicate = "duplicate" // bestaat al: niet opnieuw toegevoegd, wel in de groepen van de import
	StatusChoice    = "choice"    // het Hongaarse woord bestaat al: kies zelfde of andere betekenis
	StatusError     = "error"
)

// ImportResult is de uitkomst per regel
type ImportResult struct {
	ImportLine
	Status            string  `json:"status"`
	Message           string  `json:"message,omitempty"`
	Candidates        []Entry `json:"candidates,omitempty"`        // bij een keuze voor het Hongaarse woord: de bestaande HU->NL betekenissen
	ForwardCandidates []Entry `json:"forwardCandidates,omitempty"` // bij een keuze voor het Nederlandse woord: de bestaande NL->HU betekenissen
	Reverse           string  `json:"reverse,omitempty"`           // het Hongaarse woord waar de keuze over gaat
	// Bij een woord dat al bestaat: zo staat het er, en of het de groepen van de import erbij kreeg
	Existing string `json:"existing,omitempty"`
	Grouped  bool   `json:"grouped,omitempty"`
	// Synoniemen op de regel: wie een eigen HU->NL kaart krijgt, en wie al een andere betekenis heeft
	ExtraReverse   []string `json:"extraReverse,omitempty"`
	SkippedReverse []string `json:"skippedReverse,omitempty"`
}

// Import voegt de regels in volgorde toe aan lib. Elke regel ziet de regels erboven, zodat
// twee regels met hetzelfde Hongaarse woord ook herkend worden. Gebruik een kloon van de
// woordenschat om alleen te controleren. ok is true als er geen fouten of open keuzes zijn.
// groups gaan naar alle woorden uit de import.
func Import(lib *storage.Library, lines []ImportLine, choices map[int]ImportChoice, groups []string) (results []ImportResult, ok bool) {
	ok = true
	for _, line := range lines {
		res := importLine(lib, line, choices, groups)
		if res.Status == StatusError || res.Status == StatusChoice {
			ok = false
		}
		results = append(results, res)
	}
	return results, ok
}

func importLine(lib *storage.Library, line ImportLine, choices map[int]ImportChoice, groups []string) ImportResult {
	choice := choices[line.No]
	if choice.Context != nil {
		line.Context = strings.TrimSpace(*choice.Context)
	}
	res := ImportResult{ImportLine: line}
	fail := func(status, msg string) ImportResult {
		res.Status, res.Message = status, msg
		return res
	}
	if line.Err != "" {
		return fail(StatusError, line.Err)
	}

	nc, err := ParseInput(line.Dutch, line.Context, line.Hungarian)
	if err != nil {
		return fail(StatusError, err.Error())
	}
	nc.Example = line.Example
	nc.Groups = groups
	if existing := Existing(lib, nc); existing != nil {
		gs, err := NormalizeGroups(lib, groups)
		if err != nil {
			return fail(StatusError, err.Error())
		}
		var names []string
		for _, i := range existing {
			c := lib.Concepts[i]
			name := lib.Words[c.SourceWordID].Text
			if c.ContextSource != "" {
				name += " (" + c.ContextSource + ")"
			}
			names = append(names, name+" → "+strings.Join(TargetTexts(lib, c), ", "))
			res.Grouped = GroupExisting(lib, i, gs) || res.Grouped
		}
		res.Existing = strings.Join(names, " en ")
		return fail(StatusDuplicate, "bestaat al")
	}

	// Het Nederlandse woord bestaat al: aanvullen of nieuwe betekenis?
	if fc := ForwardCandidates(lib, nc.Dutch); len(fc) > 0 {
		res.ForwardCandidates = entriesFor(lib, fc)
		if choice.Forward == nil {
			return fail(StatusChoice, fmt.Sprintf("%q bestaat al: extra goed antwoord bij een bestaande betekenis, of een nieuwe betekenis?", nc.Dutch))
		}
		nc.ExtendForward, nc.ForwardHints = choice.Forward.Extend, choice.Forward.Hints
		if _, err := checkForward(lib, nc); err != nil {
			if errors.Is(err, ErrDuplicate) {
				return fail(StatusDuplicate, "alle Hongaarse woorden zijn al goede antwoorden bij die betekenis")
			}
			if errors.Is(err, ErrHintNeeded) || errors.Is(err, ErrContextTaken) {
				// Op te lossen in het voorbeeld zelf: daar staat een veld voor de hint van deze regel
				return fail(StatusChoice, fmt.Sprintf("geef deze nieuwe betekenis een hint die hem onderscheidt van de bestaande betekenis van %q", nc.Dutch))
			}
			return fail(StatusChoice, err.Error())
		}
	}

	// Het Hongaarse woord bestaat al: zelfde of andere betekenis?
	res.Reverse = ReverseSourceFor(lib, nc)
	if cands := ReverseCandidates(lib, res.Reverse); len(cands) > 0 {
		if !choice.Reverse {
			res.Candidates = entriesFor(lib, cands)
			return fail(StatusChoice, fmt.Sprintf("%q bestaat al: zelfde of andere betekenis?", res.Reverse))
		}
		if choice.MergeInto != "" {
			nc.MergeReverseInto = choice.MergeInto
		} else {
			nc.ExistingHints = choice.ExistingHints
			for _, i := range cands {
				c := lib.Concepts[i]
				if strings.TrimSpace(c.ContextSource) == "" && strings.TrimSpace(choice.ExistingHints[c.ID]) == "" {
					res.Candidates = entriesFor(lib, cands)
					return fail(StatusChoice, fmt.Sprintf("het bestaande %q heeft nog geen hint; geef het er een zodat de betekenissen uit elkaar te houden zijn", lib.Words[c.SourceWordID].Text))
				}
			}
		}
	}

	added, err := AddPair(lib, nc)
	if err != nil {
		return fail(StatusError, err.Error())
	}
	res.ExtraReverse, res.SkippedReverse = added.ExtraReverse, added.SkippedReverse
	res.Status = StatusAdded
	return res
}

func entriesFor(lib *storage.Library, idx []int) []Entry {
	ids := map[string]bool{}
	for _, i := range idx {
		ids[lib.Concepts[i].ID] = true
	}
	var out []Entry
	for _, e := range Entries(lib) {
		if ids[e.ID] {
			out = append(out, e)
		}
	}
	return out
}
