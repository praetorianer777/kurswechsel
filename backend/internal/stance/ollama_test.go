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

func TestOllama(t *testing.T) {
	for _, think := range []bool{false, true} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			_, hasThink := req["think"]
			if hasThink == think {
				t.Errorf("think=%v: request think field present = %v", think, hasThink)
			}
			if req["model"] != "qwen3:30b-a3b" || req["format"] == nil || req["stream"] != false {
				t.Errorf("request = %v", req)
			}
			msgs := req["messages"].([]any)
			if !strings.Contains(msgs[1].(map[string]any)["content"].(string), "Wehrpflicht") {
				t.Error("paragraph missing from user message")
			}
			w.Write([]byte(`{"message":{"role":"assistant","content":"{\"relevant\":true,\"stance\":\"dagegen\",\"quote\":\"q\",\"rationale\":\"r\",\"confidence\":0.8}"}}`))
		}))
		o := &Ollama{BaseURL: srv.URL, Model: "qwen3:30b-a3b", Think: think}
		a, err := o.Classify(context.Background(), topic.Wehrpflicht, "Keine Wehrpflicht!")
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if a.Stance != Against || a.Confidence != 0.8 {
			t.Errorf("answer = %+v", a)
		}
	}
	if (&Ollama{Model: "m"}).Name() != "ollama/m" {
		t.Error("name")
	}
}

func TestOllamaErrors(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"http error": func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "model not found", 404) },
		"bad body":   func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("<html>")) },
		"bad answer": func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"message":{"content":"ja"}}`)) },
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			if _, err := (&Ollama{BaseURL: srv.URL, Model: "m"}).Classify(context.Background(), topic.Wehrpflicht, "x"); err == nil {
				t.Error("want error")
			}
		})
	}
}
