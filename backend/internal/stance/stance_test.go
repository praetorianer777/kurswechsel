package stance

import (
	"context"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func TestParse(t *testing.T) {
	a, err := Parse(`{"relevant":true,"stance":"dafuer","quote":"q","rationale":"r","confidence":1.7}`)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Relevant || a.Stance != For || a.Confidence != 1 {
		t.Errorf("got %+v (confidence must be clamped to 1)", a)
	}
}

func TestParseLenient(t *testing.T) {
	a, err := Parse("Hier die Antwort:\n```json\n{\"relevant\":false,\"stance\":\"neutral\",\"confidence\":-1}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if a.Relevant || a.Stance != Neutral || a.Confidence != 0 {
		t.Errorf("got %+v", a)
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"dafür", `{"stance":"pro"}`, `{"stance":`, ""} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): want error", in)
		}
	}
}

func TestVerbatim(t *testing.T) {
	text := "Die Wehrpflicht bleibt\nausgesetzt, und das ist gut."
	for quote, want := range map[string]bool{
		"Wehrpflicht bleibt ausgesetzt":     true,
		"„Wehrpflicht bleibt ausgesetzt“":   true,
		"\"Wehrpflicht bleibt ausgesetzt\"": true,
		"Wehrpflicht ist ausgesetzt":        false,
		"   ":                               false,
	} {
		if got := Verbatim(text, quote); got != want {
			t.Errorf("Verbatim(%q) = %v, want %v", quote, got, want)
		}
	}
}

func TestPrompts(t *testing.T) {
	sys := SystemPrompt(topic.Wehrpflicht)
	for _, want := range []string{topic.Wehrpflicht.Definition, "VERBATIM", "German"} {
		if !strings.Contains(sys, want) {
			t.Errorf("system prompt lacks %q", want)
		}
	}
	if UserPrompt("abc") != "Paragraph:\nabc" {
		t.Error("user prompt changed")
	}
	s := Schema()
	if s["additionalProperties"] != false || len(s["required"].([]string)) != 5 {
		t.Errorf("schema = %v", s)
	}
}

func TestFake(t *testing.T) {
	ctx := context.Background()
	for text, want := range map[string]Answer{
		"Die Bundeswehr braucht Panzer.":                             {Stance: Neutral},
		"Wir müssen die Wehrpflicht wieder einführen. Das ist klar.": {Relevant: true, Stance: For, Quote: "Wir müssen die Wehrpflicht wieder einführen."},
		"Der Wehrdienst muss freiwillig bleiben.":                    {Relevant: true, Stance: Against, Quote: "Der Wehrdienst muss freiwillig bleiben."},
		"Glauben Sie wirklich, die Musterung löst das?":              {Relevant: true, Stance: Unclear, Quote: "Glauben Sie wirklich, die Musterung löst das?"},
		"Heute beraten wir das Wehrdienstgesetz in erster Lesung":    {Relevant: true, Stance: Neutral, Quote: "Heute beraten wir das Wehrdienstgesetz in erster Lesung"},
	} {
		got, err := Fake{}.Classify(ctx, topic.Wehrpflicht, text)
		if err != nil {
			t.Fatal(err)
		}
		if got.Relevant != want.Relevant || got.Stance != want.Stance || got.Quote != want.Quote {
			t.Errorf("Fake(%q) = %+v, want %+v", text, got, want)
		}
		if got.Relevant && !Verbatim(text, got.Quote) {
			t.Errorf("Fake(%q) quote is not verbatim", text)
		}
	}
	if (Fake{}).Name() != "fake" {
		t.Error("name")
	}
}
