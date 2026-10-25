package catalog

import (
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
				line.Err = "meerdere Nederlandse woorden: zet elk op een eigen regel"
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

// ImportChoice is de keuze voor een regel waarvan het Hongaarse woord al bestaat
type ImportChoice struct {
	MergeInto     string            `json:"mergeInto"`     // bestaand HU->NL concept met dezelfde betekenis; leeg = andere betekenis
	ExistingHints map[string]string `json:"existingHints"` // bij een andere betekenis: hints voor bestaande concepten zonder hint
}

// Line-statussen
const (
	StatusAdded     = "added"     // toegevoegd (of: wordt toegevoegd, bij controleren)
	StatusDuplicate = "duplicate" // bestaat al: overgeslagen
	StatusChoice    = "choice"    // het Hongaarse woord bestaat al: kies zelfde of andere betekenis
	StatusError     = "error"
)

// ImportResult is de uitkomst per regel
type ImportResult struct {
	ImportLine
	Status     string  `json:"status"`
	Message    string  `json:"message,omitempty"`
	Candidates []Entry `json:"candidates,omitempty"` // bij StatusChoice: de bestaande betekenissen
}

// Import voegt de regels in volgorde toe aan lib. Elke regel ziet de regels erboven, zodat
// twee regels met hetzelfde Hongaarse woord ook herkend worden. Gebruik een kloon van de
// woordenschat om alleen te controleren. ok is true als er geen fouten of open keuzes zijn.
func Import(lib *storage.Library, lines []ImportLine, choices map[int]ImportChoice) (results []ImportResult, ok bool) {
	ok = true
	for _, line := range lines {
		res := importLine(lib, line, choices)
		if res.Status == StatusError || res.Status == StatusChoice {
			ok = false
		}
		results = append(results, res)
	}
	return results, ok
}

func importLine(lib *storage.Library, line ImportLine, choices map[int]ImportChoice) ImportResult {
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
	if IsDuplicate(lib, nc) {
		return fail(StatusDuplicate, "bestaat al met deze context")
	}

	if cands := ReverseCandidates(lib, nc.Hungarian[0]); len(cands) > 0 {
		choice, chosen := choices[line.No]
		if !chosen {
			res.Candidates = entriesFor(lib, cands)
			return fail(StatusChoice, fmt.Sprintf("%q bestaat al: zelfde of andere betekenis?", nc.Hungarian[0]))
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

	if _, err := AddPair(lib, nc); err != nil {
		return fail(StatusError, err.Error())
	}
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
