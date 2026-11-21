// De woorden voor de -i-regel. -i maakt van een plaats of tijd een bijvoeglijk naamwoord
// (budapesti, reggeli). Achter andere zelfstandige naamwoorden klopt het meestal niet (kávéi is
// "zijn koffies", házi is "zelfgemaakt"), dus de regel oefent alleen met deze lijst. De plaatsen
// hoef je niet als woord te leren: de Nederlandse naam staat bij de opgave.

package grammar

import "strings"

type iWord struct {
	hu, nl string
	form   string // alleen als -i er niet gewoon achter komt (este -> esti)
}

var iWords = []iWord{
	// Plaatsen
	{hu: "Budapest", nl: "Boedapest"}, {hu: "Győr", nl: "Győr"}, {hu: "Szeged", nl: "Szeged"},
	{hu: "Debrecen", nl: "Debrecen"}, {hu: "Pécs", nl: "Pécs"}, {hu: "Miskolc", nl: "Miskolc"},
	{hu: "Eger", nl: "Eger", form: "egri"}, {hu: "Balaton", nl: "Balatonmeer"}, {hu: "Duna", nl: "Donau"},
	{hu: "Magyarország", nl: "Hongarije"}, {hu: "Hollandia", nl: "Nederland"}, {hu: "Németország", nl: "Duitsland"},
	{hu: "Ausztria", nl: "Oostenrijk"}, {hu: "Amerika", nl: "Amerika"}, {hu: "Európa", nl: "Europa"},
	{hu: "Amszterdam", nl: "Amsterdam"}, {hu: "Utrecht", nl: "Utrecht"}, {hu: "Rotterdam", nl: "Rotterdam"},
	{hu: "London", nl: "Londen"}, {hu: "Berlin", nl: "Berlijn"}, {hu: "Bécs", nl: "Wenen"},
	{hu: "Párizs", nl: "Parijs"}, {hu: "Róma", nl: "Rome"}, {hu: "Prága", nl: "Praag"},
	// Tijd
	{hu: "reggel", nl: "ochtend"}, {hu: "este", nl: "avond", form: "esti"}, {hu: "éjszaka", nl: "nacht"},
	{hu: "délelőtt", nl: "voormiddag"}, {hu: "délután", nl: "namiddag"}, {hu: "nap", nl: "dag (24 uur)"},
	{hu: "hét", nl: "week", form: "heti"}, {hu: "hónap", nl: "maand", form: "havi"}, {hu: "év", nl: "jaar"},
	{hu: "tavasz", nl: "lente"}, {hu: "nyár", nl: "zomer"}, {hu: "ősz", nl: "herfst"}, {hu: "tél", nl: "winter"},
	{hu: "hétfő", nl: "maandag"}, {hu: "kedd", nl: "dinsdag"}, {hu: "szerda", nl: "woensdag"},
	{hu: "csütörtök", nl: "donderdag"}, {hu: "péntek", nl: "vrijdag"}, {hu: "szombat", nl: "zaterdag"},
	{hu: "vasárnap", nl: "zondag"}, {hu: "hétvége", nl: "weekend"},
}

// poolOf maakt van een vaste lijst opgavewoorden Word's; Known als het woord ook in je eigen
// lijst staat. De Nederlandse omschrijving komt uit de lijst (twee (uur) bij -kor, niet de hint
// van je kaart).
func poolOf(list []iWord, key string, words []Word) []Word {
	own := map[string]Word{}
	for _, w := range words {
		own[strings.ToLower(w.Hungarian)] = w
	}
	out := make([]Word, 0, len(list))
	for _, iw := range list {
		w := Word{Hungarian: iw.hu, Dutch: iw.nl, Type: "zn"}
		if mine, ok := own[strings.ToLower(iw.hu)]; ok {
			w.Known = mine.Known
		}
		if iw.form != "" {
			w.Forms = map[string]string{key: iw.form}
		}
		out = append(out, w)
	}
	return out
}
