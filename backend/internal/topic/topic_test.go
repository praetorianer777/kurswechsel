package topic

import "testing"

func TestGet(t *testing.T) {
	if got, ok := Get("wehrpflicht"); !ok || got.Name != "Wehrpflicht" {
		t.Errorf("Get(wehrpflicht) = %+v, %v", got, ok)
	}
	if _, ok := Get("nope"); ok {
		t.Error("Get(nope) found something")
	}
}

func TestWehrpflichtKeywords(t *testing.T) {
	for text, want := range map[string]bool{
		"Die Wehrpflicht muss zurück.":                              true,
		"Der neue Wehrdienst ist freiwillig.":                       true,
		"Ab 2027 gibt es wieder eine Musterung.":                    true,
		"Ein verpflichtendes Gesellschaftsjahr für alle.":           true,
		"Niemand darf zum Dienst an der Waffe verpflichtet werden.": true,
		"Die Bundeswehr braucht mehr Panzer.":                       false,
	} {
		if got := Wehrpflicht.Keywords.MatchString(text); got != want {
			t.Errorf("Keywords(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestSlugsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, tp := range All {
		if seen[tp.Slug] {
			t.Errorf("duplicate slug %q", tp.Slug)
		}
		seen[tp.Slug] = true
		if tp.Keywords == nil || tp.Definition == "" || tp.Question == "" {
			t.Errorf("topic %q incomplete", tp.Slug)
		}
	}
}
