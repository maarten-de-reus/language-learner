package grammar

import "testing"

func TestPossessive(t *testing.T) {
	for word, want := range map[string][6]string{
		"alma":      {"almám", "almád", "almája", "almánk", "almátok", "almájuk"},
		"kefe":      {"kefém", "keféd", "keféje", "kefénk", "kefétek", "keféjük"},
		"kávé":      {"kávém", "kávéd", "kávéja", "kávénk", "kávétok", "kávéjuk"},
		"autó":      {"autóm", "autód", "autója", "autónk", "autótok", "autójuk"},
		"cipő":      {"cipőm", "cipőd", "cipője", "cipőnk", "cipőtök", "cipőjük"},
		"ház":       {"házam", "házad", "háza", "házunk", "házatok", "házuk"},
		"ló":        {"lovam", "lovad", "lova", "lovunk", "lovatok", "lovuk"},
		"kéz":       {"kezem", "kezed", "keze", "kezünk", "kezetek", "kezük"},
		"sarok":     {"sarkom", "sarkod", "sarka", "sarkunk", "sarkotok", "sarkuk"},
		"tükör":     {"tükröm", "tükröd", "tükre", "tükrünk", "tükrötök", "tükrük"},
		"könyv":     {"könyvem", "könyved", "könyve", "könyvünk", "könyvetek", "könyvük"},
		"föld":      {"földem", "földed", "földje", "földünk", "földetek", "földjük"},
		"kert":      {"kertem", "kerted", "kertje", "kertünk", "kertetek", "kertjük"},
		"bolt":      {"boltom", "boltod", "boltja", "boltunk", "boltotok", "boltjuk"},
		"pénz":      {"pénzem", "pénzed", "pénze", "pénzünk", "pénzetek", "pénzük"},
		"hegy":      {"hegyem", "hegyed", "hegye", "hegyünk", "hegyetek", "hegyük"},
		"lány":      {"lányom", "lányod", "lánya", "lányunk", "lányotok", "lányuk"},
		"orvos":     {"orvosom", "orvosod", "orvosa", "orvosunk", "orvosotok", "orvosuk"},
		"gyümölcs":  {"gyümölcsöm", "gyümölcsöd", "gyümölcse", "gyümölcsünk", "gyümölcsötök", "gyümölcsük"},
		"feladat":   {"feladatom", "feladatod", "feladata", "feladatunk", "feladatotok", "feladatuk"},
		"fiú":       {"fiam", "fiad", "fia", "fiunk", "fiatok", "fiuk"},
		"anya":      {"anyám", "anyád", "anyja", "anyánk", "anyátok", "anyjuk"},
		"kórház":    {"kórházam", "kórházad", "kórháza", "kórházunk", "kórházatok", "kórházuk"},
		"játszótér": {"játszóterem", "játszótered", "játszótere", "játszóterünk", "játszóteretek", "játszóterük"},
		"fog":       {"fogam", "fogad", "foga", "fogunk", "fogatok", "foguk"},
		"szombat":   {"szombatom", "szombatod", "szombatja", "szombatunk", "szombatotok", "szombatjuk"},
		"perc":      {"percem", "perced", "perce", "percünk", "percetek", "percük"},
		"telefon":   {"telefonom", "telefonod", "telefonja", "telefonunk", "telefonotok", "telefonjuk"},
		"tankönyv":  {"tankönyvem", "tankönyved", "tankönyve", "tankönyvünk", "tankönyvetek", "tankönyvük"},
	} {
		for p, form := range want {
			if f, ok := Possessive(word, Person(p)); !ok || f.Text != form {
				t.Errorf("%s bezittelijk, %s: %q (ok %v), verwacht %q; %v", word, PersonLabels[p], f.Text, ok, form, f.Steps)
			}
		}
	}
	// asztala maar telefonja: zonder lijst niet zeker, dan alleen ő/ők niet gevraagd (tányér)
	if f, ok := Possessive("tányér", Sg1); !ok || f.Text != "tányérom" {
		t.Errorf("tányérom: %+v %v", f, ok)
	}
	if f, ok := Possessive("tányér", Sg3); ok {
		t.Errorf("tányér, ő: niet zeker zonder lijst, kreeg %s", f.Text)
	}
	if _, ok := Possessive("kar", Sg1); ok {
		t.Error("kar: kort woord, niet in de lijsten")
	}
}
