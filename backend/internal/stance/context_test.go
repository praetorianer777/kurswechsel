package stance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func TestContextUserPrompt(t *testing.T) {
	c := Context{PreviousSpeaker: "Max Frager", Previous: "Wollen Sie die Wehrpflicht?", Before: []string{"A.", "B."}, After: []string{"C."}}
	got := ContextUserPrompt("Nein, niemals.", c)
	want := "Context – end of the previous speech (Max Frager):\nWollen Sie die Wehrpflicht?\n\n" +
		"Context – earlier in the same speech:\nA.\n\nB.\n\n" +
		"Paragraph to classify:\nNein, niemals.\n\n" +
		"Context – later in the same speech:\nC."
	if got != want {
		t.Errorf("prompt =\n%s\nwant\n%s", got, want)
	}
	if ContextUserPrompt("X.", Context{}) != "Paragraph to classify:\nX." {
		t.Error("empty context")
	}
	if !(Context{}).Empty() || c.Empty() {
		t.Error("Empty")
	}
	if !strings.HasPrefix(ContextSystemPrompt(topic.Wehrpflicht), SystemPrompt(topic.Wehrpflicht)) {
		t.Error("context system prompt must extend v1")
	}
}

func TestOllamaInContext(t *testing.T) {
	var got []map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []map[string]string `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		got = req.Messages
		w.Write([]byte(`{"message":{"content":"{\"relevant\":true,\"stance\":\"dagegen\",\"quote\":\"niemals\",\"rationale\":\"r\",\"confidence\":0.9}"}}`))
	}))
	defer srv.Close()
	o := &Ollama{BaseURL: srv.URL, Model: "m"}

	if _, err := o.ClassifyInContext(context.Background(), topic.Wehrpflicht, "Nein, niemals.", Context{Previous: "Wehrpflicht?"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got[0]["content"], "Use the context only to understand") || !strings.Contains(got[1]["content"], "Paragraph to classify:") {
		t.Errorf("context not sent: %v", got)
	}
	if _, err := o.Classify(context.Background(), topic.Wehrpflicht, "Nein."); err != nil {
		t.Fatal(err)
	}
	if got[0]["content"] != SystemPrompt(topic.Wehrpflicht) || got[1]["content"] != UserPrompt("Nein.") {
		t.Error("without context the request must be prompt v1 unchanged")
	}
}
