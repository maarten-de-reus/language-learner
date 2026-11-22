package grammar

import "testing"

func TestLak(t *testing.T) {
	for verb, want := range map[string]string{
		"lát": "látlak", "szeret": "szeretlek", "vár": "várlak", "hív": "hívlak", "ír": "írlak", "olvas": "olvaslak",
		"néz": "nézlek", "köszön": "köszönlek", "keres": "kereslek", "tanít": "tanítalak", "segít": "segítelek",
		"ért": "értelek", "mond": "mondalak", "küld": "küldelek", "tart": "tartalak", "hall": "hallak", "visz": "viszlek",
		"felhív": "felhívlak", "megért": "megértelek",
	} {
		if f, ok := Lak(verb); !ok || f.Text != want {
			t.Errorf("%s: %q (ok %v), verwacht %q", verb, f.Text, ok, want)
		}
	}
	for _, verb := range []string{"megy", "lakik", "van", "sétál"} {
		if f, ok := Lak(verb); ok {
			t.Errorf("%s heeft geen -lak-vorm, kreeg %s", verb, f.Text)
		}
	}
}
