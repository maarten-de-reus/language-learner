package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"hungarian-language-learner/internal/grammar"
	"hungarian-language-learner/internal/storage"
)

// ImportLine is één regel uit een geplakte lijst: hongaars;nederlands;context;voorbeeldzin;woordsoort;vormen
type ImportLine struct {
	No        int    `json:"no"` // regelnummer in de geplakte tekst (vanaf 1)
	Hungarian string `json:"hungarian"`
	Dutch     string `json:"dutch"`
	Context   string `json:"context"`
	Example   string `json:"example"`
	WordType  string `json:"wordType,omitempty"` // code, na controle (zie ParseWordType)
	Forms     string `json:"forms,omitempty"`    // afwijkende vormen, bijv. "t=házat, k=házak" (zie grammar.ParseForms)
	Err       string `json:"error,omitempty"`    // fout bij het lezen van de regel
}

// ParseImport leest een geplakte lijst. Kolommen worden gescheiden door ; (spaties eromheen
// mogen, hoeven niet). Context, voorbeeldzin, woordsoort en afwijkende vormen zijn optioneel. Lege regels, regels die met #
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
		case len(fields) > 6:
			line.Err = "te veel kolommen: maximaal hongaars;nederlands;context;voorbeeldzin;woordsoort;vormen (een ; in de voorbeeldzin kan niet)"
		}
		if line.Err == "" {
			fields = append(fields, "", "", "", "")
			line.Hungarian, line.Dutch, line.Context, line.Example = fields[0], fields[1], fields[2], fields[3]
			wordType, err := ParseWordType(fields[4])
			line.WordType = wordType
			forms, formErr := grammar.ParseForms(fields[5])
			line.Forms = grammar.FormatForms(forms)
			switch {
			case err != nil:
				line.Err = err.Error()
			case formErr != nil:
				line.Err = formErr.Error()
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

	Skip bool `json:"skip,omitempty"` // deze regel niet importeren (bijv. later naar kijken)
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
	StatusSkipped   = "skipped" // door jou overgeslagen: niet geïmporteerd, blokkeert de import niet
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
	Typed    bool   `json:"typed,omitempty"` // kreeg de woordsoort en/of afwijkende vormen van de import (had er nog geen)
	// Synoniemen op de regel: wie een eigen HU->NL kaart krijgt, en wie al een andere betekenis heeft
	ExtraReverse   []string `json:"extraReverse,omitempty"`
	SkippedReverse []string `json:"skippedReverse,omitempty"`
	// Note: een opmerking die de import niet tegenhoudt, bijv. een werkwoord zonder zijn partner
	Note string `json:"note,omitempty"`
}

// Import voegt de regels in volgorde toe aan lib. Elke regel ziet de regels erboven, zodat
// twee regels met hetzelfde Hongaarse woord ook herkend worden. Gebruik een kloon van de
// woordenschat om alleen te controleren. ok is true als er geen fouten of open keuzes zijn.
// De opties (groepen, alvast leren) gelden voor alle woorden uit de import.
// ImportOptions gelden voor alle regels van een import
type ImportOptions struct {
	Groups  []string // voor alle woorden uit de import
	Release bool     // nieuwe woorden alvast in de les, ook al zijn ze nog niet goedgekeurd
}

func Import(lib *storage.Library, lines []ImportLine, choices map[int]ImportChoice, opts ImportOptions) (results []ImportResult, ok bool) {
	ok = true
	for _, line := range lines {
		res := importLine(lib, line, choices, opts)
		if res.Status == StatusError || res.Status == StatusChoice {
			ok = false
		}
		results = append(results, res)
	}
	noteVerbPartners(lib, results)
	return results, ok
}

// noteVerbPartners: van een werkwoord leer je de infinitief én de 3e persoon enkelvoud (de regels
// vervoegen vanuit de 3e persoon). Komt er een nieuw werkwoord binnen zonder zijn partner, in de
// import of al in je lijst, dan zegt de import dat.
func noteVerbPartners(lib *storage.Library, results []ImportResult) {
	verbs := map[string]bool{}
	var thirds []string
	for _, c := range lib.Concepts {
		if c.WordType != "ww" || lib.Words[c.SourceWordID].Language != "nl" {
			continue
		}
		for _, id := range c.TargetWordIDs {
			hu := strings.ToLower(strings.TrimSpace(lib.Words[id].Text))
			verbs[hu] = true
			if !strings.HasSuffix(hu, "ni") {
				thirds = append(thirds, hu)
			}
		}
	}
	infinitiveOf := func(third string) string {
		if f, ok := grammar.Infinitive(third); ok {
			return f.Text
		}
		return ""
	}
	for i, r := range results {
		if r.Status != StatusAdded || r.WordType != "ww" {
			continue
		}
		hu := strings.ToLower(strings.TrimSpace(strings.Split(r.Hungarian, ",")[0]))
		if strings.HasSuffix(hu, "ni") {
			if !slices.ContainsFunc(thirds, func(t string) bool { return infinitiveOf(t) == hu }) {
				results[i].Note = "werkwoord zonder 3e persoon enkelvoud: voeg die ook toe, anders kunnen de regels het niet vervoegen"
			}
			continue
		}
		forms, _ := grammar.ParseForms(r.Forms)
		inf := forms["inf"]
		if inf == "" {
			inf = infinitiveOf(hu)
		}
		switch {
		case inf == "" && verbs[hu+"ni"]:
			// niet uit te rekenen (lő), maar lőni staat er al
		case inf == "":
			results[i].Note = "werkwoord: voeg ook de infinitief toe"
		case !verbs[inf]:
			results[i].Note = "werkwoord zonder infinitief: voeg ook " + inf + " toe"
		}
	}
}

func importLine(lib *storage.Library, line ImportLine, choices map[int]ImportChoice, opts ImportOptions) ImportResult {
	choice := choices[line.No]
	if choice.Context != nil {
		line.Context = strings.TrimSpace(*choice.Context)
	}
	res := ImportResult{ImportLine: line}
	fail := func(status, msg string) ImportResult {
		res.Status, res.Message = status, msg
		return res
	}
	if choice.Skip {
		return fail(StatusSkipped, "overgeslagen")
	}
	if line.Err != "" {
		return fail(StatusError, line.Err)
	}

	nc, err := ParseInput(line.Dutch, line.Context, line.Hungarian)
	if err != nil {
		return fail(StatusError, err.Error())
	}
	nc.Example = line.Example
	nc.Groups = opts.Groups
	nc.Release = opts.Release
	nc.WordType = line.WordType
	nc.Forms = line.Forms
	if existing := Existing(lib, nc); existing != nil {
		gs, err := NormalizeGroups(lib, opts.Groups)
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
			// Een bestaand woord zonder woordsoort krijgt die van de import; een andere laat het staan
			res.Typed = setWordType(lib, i, line.WordType, true) || res.Typed
			if forms, _ := grammar.ParseForms(line.Forms); setForms(lib, i, forms, true) {
				res.Typed = true
			}
		}
		res.Existing = strings.Join(names, " en ")
		return fail(StatusDuplicate, "bestaat al")
	}

	// Het Nederlandse woord bestaat al: aanvullen of nieuwe betekenis?
	if fc := ForwardCandidates(lib, nc.Dutch); len(fc) > 0 {
		res.ForwardCandidates = entriesOf(lib, fc)
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
			res.Candidates = entriesOf(lib, cands)
			return fail(StatusChoice, fmt.Sprintf("%q bestaat al: zelfde of andere betekenis?", res.Reverse))
		}
		if choice.MergeInto != "" {
			nc.MergeReverseInto = choice.MergeInto
		} else {
			nc.ExistingHints = choice.ExistingHints
			for _, i := range cands {
				c := lib.Concepts[i]
				if strings.TrimSpace(c.ContextSource) == "" && strings.TrimSpace(choice.ExistingHints[c.ID]) == "" {
					res.Candidates = entriesOf(lib, cands)
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
