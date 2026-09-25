package stance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// A Messages API response in the shape the API returns; no request in this
// file reaches the real API.
const recordedReply = `{
  "id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-5",
  "content": [{"type": "text", "text": "{\"relevant\":true,\"stance\":\"dafuer\",\"quote\":\"Wehrpflicht zurück\",\"rationale\":\"Fordert die Rückkehr.\",\"confidence\":0.9}"}],
  "stop_reason": "end_turn", "stop_sequence": null,
  "usage": {"input_tokens": 420, "output_tokens": 60}
}`

func claudeAgainst(t *testing.T, h http.HandlerFunc) *Claude {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Claude{
		Client: anthropic.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"), option.WithMaxRetries(0)),
		Model:  "claude-opus-5",
	}
}

func TestClaude(t *testing.T) {
	c := claudeAgainst(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		oc, _ := req["output_config"].(map[string]any)
		format, _ := oc["format"].(map[string]any)
		if format["type"] != "json_schema" || format["schema"] == nil || oc["effort"] != "low" {
			t.Errorf("output_config = %v", req["output_config"])
		}
		if req["model"] != "claude-opus-5" {
			t.Errorf("model = %v", req["model"])
		}
		sys, _ := json.Marshal(req["system"])
		if !strings.Contains(string(sys), "German Bundestag") {
			t.Errorf("system = %s", sys)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(recordedReply))
	})
	a, err := c.Classify(context.Background(), topic.Wehrpflicht, "Wir wollen die Wehrpflicht zurück.")
	if err != nil {
		t.Fatal(err)
	}
	if a.Stance != For || a.Quote != "Wehrpflicht zurück" {
		t.Errorf("answer = %+v", a)
	}
	if c.Name() != "claude/claude-opus-5" {
		t.Error("name")
	}
}

func TestClaudeRefusalAndErrors(t *testing.T) {
	refusal := strings.Replace(recordedReply, `"end_turn"`, `"refusal"`, 1)
	for name, h := range map[string]http.HandlerFunc{
		"refusal": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(refusal))
		},
		"rate limited": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`))
		},
		"no text": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"m","type":"message","role":"assistant","model":"claude-opus-5","content":[],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":0}}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := claudeAgainst(t, h).Classify(context.Background(), topic.Wehrpflicht, "x"); err == nil {
				t.Error("want error")
			}
		})
	}
}
