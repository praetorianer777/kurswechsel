// Package ollama is a minimal client for the two Ollama endpoints the spikes
// use: /api/embed and /api/chat with a JSON schema.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client talks to one Ollama server.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client for baseURL, e.g. http://127.0.0.1:11434.
func New(baseURL string) *Client { return &Client{BaseURL: baseURL, HTTP: http.DefaultClient} }

// Embed returns one embedding per input.
func (c *Client) Embed(ctx context.Context, model string, input []string) ([][]float32, error) {
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	err := c.post(ctx, "/api/embed", map[string]any{"model": model, "input": input}, &out)
	if err != nil {
		return nil, err
	}
	if len(out.Embeddings) != len(input) {
		return nil, fmt.Errorf("embed: got %d vectors for %d inputs", len(out.Embeddings), len(input))
	}
	return out.Embeddings, nil
}

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResult is the model's reply plus Ollama's timing.
type ChatResult struct {
	Content      string
	TotalNanos   int64
	PromptTokens int
	OutputTokens int
}

// ChatJSON asks for a reply that conforms to schema, deterministically.
// With think false, thinking is switched off explicitly; with think true the
// model's default applies. Ollama 0.30 drops the schema for thinking models
// when thinking is switched off, so those have to be run with think true.
func (c *Client) ChatJSON(ctx context.Context, model string, msgs []Message, schema any, think bool) (ChatResult, error) {
	req := map[string]any{
		"model":    model,
		"messages": msgs,
		"format":   schema,
		"stream":   false,
		"options":  map[string]any{"temperature": 0, "seed": 1, "num_ctx": 8192},
	}
	if !think {
		req["think"] = false
	}
	var out struct {
		Message         Message `json:"message"`
		TotalDuration   int64   `json:"total_duration"`
		PromptEvalCount int     `json:"prompt_eval_count"`
		EvalCount       int     `json:"eval_count"`
	}
	if err := c.post(ctx, "/api/chat", req, &out); err != nil {
		return ChatResult{}, err
	}
	return ChatResult{Content: out.Message.Content, TotalNanos: out.TotalDuration, PromptTokens: out.PromptEvalCount, OutputTokens: out.EvalCount}, nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("POST %s: %s: %s", path, resp.Status, bytes.TrimSpace(msg))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
