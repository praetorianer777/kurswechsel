package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"embeddings":[[1,0],[0,1]]}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL).Embed(context.Background(), "bge-m3", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1][1] != 1 {
		t.Errorf("got %v", got)
	}
}

func TestEmbedCountMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"embeddings":[[1,0]]}`))
	}))
	defer srv.Close()

	if _, err := New(srv.URL).Embed(context.Background(), "m", []string{"a", "b"}); err == nil {
		t.Fatal("want error")
	}
}

func TestChatJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["think"] != false || req["stream"] != false || req["format"] == nil {
			t.Errorf("unexpected request: %v", req)
		}
		w.Write([]byte(`{"message":{"role":"assistant","content":"{\"stance\":\"neutral\"}"},"total_duration":5,"prompt_eval_count":10,"eval_count":3}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL).ChatJSON(context.Background(), "m", []Message{{Role: "user", Content: "x"}}, map[string]any{"type": "object"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != `{"stance":"neutral"}` || got.TotalNanos != 5 || got.PromptTokens != 10 || got.OutputTokens != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestChatJSONThinkOmitted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if _, ok := req["think"]; ok {
			t.Errorf("think must be omitted, got %v", req["think"])
		}
		w.Write([]byte(`{"message":{"content":"{}"}}`))
	}))
	defer srv.Close()

	if _, err := New(srv.URL).ChatJSON(context.Background(), "m", nil, nil, true); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := New(srv.URL).ChatJSON(context.Background(), "m", nil, nil, false)
	if err == nil || err.Error() != "POST /api/chat: 404 Not Found: model not found" {
		t.Fatalf("err = %v", err)
	}
}
