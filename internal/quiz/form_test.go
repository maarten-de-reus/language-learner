package quiz

import (
	"path/filepath"
	"testing"

	"hungarian-language-learner/internal/storage"
)

func TestFormOfWrongAnswer(t *testing.T) {
	words := map[string]storage.Word{
		"nl_leest": {ID: "nl_leest", Language: "nl", Text: "leest"}, "hu_olvas": {ID: "hu_olvas", Language: "hu", Text: "olvas"},
		"nl_lezen": {ID: "nl_lezen", Language: "nl", Text: "lezen"}, "hu_olvasni": {ID: "hu_olvasni", Language: "hu", Text: "olvasni"},
		"nl_huis": {ID: "nl_huis", Language: "nl", Text: "huis"}, "hu_haz": {ID: "hu_haz", Language: "hu", Text: "ház"},
	}
	concepts := []storage.TranslationConcept{
		{ID: "c_leest", SourceWordID: "nl_leest", TargetWordIDs: []string{"hu_olvas"}, WordType: "ww"},
		{ID: "c_lezen", SourceWordID: "nl_lezen", TargetWordIDs: []string{"hu_olvasni"}, WordType: "ww"},
		{ID: "c_huis", SourceWordID: "nl_huis", TargetWordIDs: []string{"hu_haz"}, WordType: "zn"},
		{ID: "c_haz", SourceWordID: "hu_haz", TargetWordIDs: []string{"nl_huis"}, WordType: "zn"},
	}
	s := New(&storage.Library{Words: words, Concepts: concepts}, filepath.Join(t.TempDir(), "voortgang.json"), 10)
	for _, c := range []struct{ id, input, rule, person string }{
		{"c_leest", "olvasok", "ww-onbep", "én (ik)"},
		{"c_leest", "olvasni", "ww-inf", ""},
		{"c_lezen", "olvas", "ww-onbep", ""}, // de 3e persoon bij een infinitiefkaart
		{"c_lezen", "olvasunk", "ww-onbep", "mi (wij)"},
		{"c_huis", "házban", "ban", ""},
	} {
		res, _ := s.Practice(c.id, c.input)
		if res.Correct || res.Form == nil || res.Form.Rule != c.rule || res.Form.Person != c.person {
			t.Errorf("%s / %s: %+v %+v", c.id, c.input, res, res.Form)
		}
	}
	if res, _ := s.Practice("c_huis", "kert"); res.Form != nil {
		t.Errorf("kert is geen vorm van ház: %+v", res.Form)
	}
	if res, _ := s.Practice("c_haz", "huizen"); res.Form != nil {
		t.Error("naar het Nederlands geen vormherkenning")
	}
}
