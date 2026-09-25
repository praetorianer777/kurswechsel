package stance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// Ollama classifies with a local model through Ollama's chat API.
type Ollama struct {
	BaseURL string
	Model   string
	// Think leaves the model's thinking on. Ollama 0.30 drops the JSON schema
	// for thinking models when thinking is switched off, so those need it.
	Think bool
	HTTP  *http.Client
}

// Name implements Classifier.
func (o *Ollama) Name() string { return "ollama/" + o.Model }

// Classify implements Classifier.
func (o *Ollama) Classify(ctx context.Context, t topic.Topic, paragraph string) (Answer, error) {
	req := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{"role": "system", "content": SystemPrompt(t)},
			{"role": "user", "content": UserPrompt(paragraph)},
		},
		"format":  Schema(),
		"stream":  false,
		"options": map[string]any{"temperature": 0, "seed": 1, "num_ctx": 8192},
	}
	if !o.Think {
		req["think"] = false
	}
	body, err := json.Marshal(req)
	if err != nil {
		return Answer{}, err
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Answer{}, err
	}
	hr.Header.Set("Content-Type", "application/json")
	client := o.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(hr)
	if err != nil {
		return Answer{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Answer{}, fmt.Errorf("ollama: %s: %s", resp.Status, bytes.TrimSpace(msg))
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Answer{}, fmt.Errorf("ollama: %w", err)
	}
	return Parse(out.Message.Content)
}
