package stance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/spikes/gold"
	"github.com/praetorianer777/kurswechsel/spikes/ollama"
)

type fakeChat struct {
	reply string
	err   error
	got   []ollama.Message
	think bool
}

func (f *fakeChat) ChatJSON(_ context.Context, _ string, msgs []ollama.Message, _ any, think bool) (ollama.ChatResult, error) {
	f.got = msgs
	f.think = think
	return ollama.ChatResult{Content: f.reply, TotalNanos: 42, OutputTokens: 7}, f.err
}

func TestClassify(t *testing.T) {
	it := gold.Item{ID: "1", Text: "Wir müssen die  Wehrpflicht wieder einführen."}
	fc := &fakeChat{reply: `{"relevant":true,"stance":"dafuer","quote":"„die Wehrpflicht wieder einführen“","rationale":"Fordert die Wiedereinführung.","confidence":0.9}`}

	r, err := Classify(context.Background(), fc, Model{Name: "m", Think: true}, it)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Relevant || r.Stance != gold.For || !r.QuoteVerbatim || r.Nanos != 42 || r.OutputTokens != 7 {
		t.Errorf("got %+v", r)
	}
	if !fc.think {
		t.Error("think flag not passed through")
	}
	if len(fc.got) != 2 || fc.got[0].Content != System || !strings.Contains(fc.got[1].Content, it.Text) {
		t.Errorf("prompt = %+v", fc.got)
	}
}

func TestClassifyErrors(t *testing.T) {
	it := gold.Item{ID: "1", Text: "x"}
	for name, fc := range map[string]*fakeChat{
		"transport":     {err: errors.New("boom")},
		"not json":      {reply: "dafür"},
		"unknown label": {reply: `{"relevant":true,"stance":"pro"}`},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Classify(context.Background(), fc, Model{Name: "m"}, it); err == nil {
				t.Error("want error")
			}
		})
	}
}

func TestParseLenient(t *testing.T) {
	a, err := Parse("Hier ist die Antwort:\n```json\n{\"relevant\":false,\"stance\":\"neutral\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if a.Relevant || a.Stance != "neutral" {
		t.Errorf("got %+v", a)
	}
}

func TestParseModel(t *testing.T) {
	if m := ParseModel("qwen3.5:9b@think"); m.Name != "qwen3.5:9b" || !m.Think || m.String() != "qwen3.5:9b (thinking)" {
		t.Errorf("got %+v", m)
	}
	if m := ParseModel("llama3.1:8b"); m.Name != "llama3.1:8b" || m.Think || m.String() != "llama3.1:8b" {
		t.Errorf("got %+v", m)
	}
}

func TestVerbatim(t *testing.T) {
	text := "Die Wehrpflicht bleibt\nausgesetzt, und das ist gut."
	for quote, want := range map[string]bool{
		"Wehrpflicht bleibt ausgesetzt":     true,
		"\"Wehrpflicht bleibt ausgesetzt\"": true,
		"Wehrpflicht ist ausgesetzt":        false,
		"   ":                               false,
	} {
		if got := Verbatim(text, quote); got != want {
			t.Errorf("Verbatim(%q) = %v, want %v", quote, got, want)
		}
	}
}
