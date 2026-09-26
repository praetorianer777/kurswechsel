package stance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func TestSentences(t *testing.T) {
	for in, want := range map[string][]string{
		"Eins. Zwei! Drei?": {"Eins.", "Zwei!", "Drei?"},
		"Wie Dr. Müller sagt, z. B. die Musterung. Gut so.": {"Wie Dr. Müller sagt, z. B. die Musterung.", "Gut so."},
		"Er sagte: „Nein.“ Danach Stille.":                  {"Er sagte: „Nein.“", "Danach Stille."},
		"Kosten von 3 Mrd. Euro. Äußerst teuer.":            {"Kosten von 3 Mrd. Euro.", "Äußerst teuer."},
		"ohne Punkt am Ende":                                {"ohne Punkt am Ende"},
		"Am 1. Januar beginnt es. Dann":                     {"Am 1. Januar beginnt es.", "Dann"},
	} {
		if got := Sentences(in); !reflect.DeepEqual(got, want) {
			t.Errorf("Sentences(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestQuoteFor(t *testing.T) {
	p := "Wir reden heute über die Bundeswehr. Die Wehrpflicht muss wieder her. Danke."
	if got := QuoteFor(p, topic.Wehrpflicht); got != "Die Wehrpflicht muss wieder her." {
		t.Errorf("QuoteFor = %q", got)
	}
	if !Verbatim(p, QuoteFor(p, topic.Wehrpflicht)) {
		t.Error("quote not verbatim")
	}
	if got := QuoteFor(" Nichts zum Thema. ", topic.Wehrpflicht); got != "Nichts zum Thema." {
		t.Errorf("fallback = %q", got)
	}
}

func TestOllamaLabelStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		props := req["format"].(map[string]any)["properties"].(map[string]any)
		if len(props) != 2 {
			t.Errorf("label schema has %d properties", len(props))
		}
		sys := req["messages"].([]any)[0].(map[string]any)["content"].(string)
		if !strings.Contains(sys, `"relevant" and "stance" only`) {
			t.Errorf("system prompt = %q", sys)
		}
		w.Write([]byte(`{"message":{"content":"{\"relevant\":true,\"stance\":\"dafuer\"}"}}`))
	}))
	defer srv.Close()
	o := &Ollama{BaseURL: srv.URL, Model: "m", Style: StyleLabel}
	a, err := o.Classify(context.Background(), topic.Wehrpflicht, "Einleitung. Die Wehrpflicht muss zurück.")
	if err != nil {
		t.Fatal(err)
	}
	if a.Stance != For || a.Quote != "Die Wehrpflicht muss zurück." || a.Rationale != "" {
		t.Errorf("answer = %+v", a)
	}
	if o.Name() != "ollama/m+label" || o.Prompt() != LabelPromptVersion || (&Ollama{Model: "m"}).Prompt() != PromptVersion {
		t.Error("name or prompt version")
	}
}
