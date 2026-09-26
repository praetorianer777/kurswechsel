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
	// Style is StyleFull (the default when empty) or StyleLabel.
	Style string
	HTTP  *http.Client
}

// Name implements Classifier. The label-only style is part of the name so
// its stored results are never mistaken for full-prompt ones.
func (o *Ollama) Name() string {
	if o.Style == StyleLabel {
		return "ollama/" + o.Model + "+label"
	}
	return "ollama/" + o.Model
}

// Prompt reports the prompt version this classifier uses.
func (o *Ollama) Prompt() string {
	if o.Style == StyleLabel {
		return LabelPromptVersion
	}
	return PromptVersion
}

// Classify implements Classifier.
func (o *Ollama) Classify(ctx context.Context, t topic.Topic, paragraph string) (Answer, error) {
	system, schema := SystemPrompt(t), Schema()
	if o.Style == StyleLabel {
		system, schema = LabelSystemPrompt(t), LabelSchema()
	}
	req := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": UserPrompt(paragraph)},
		},
		"format":  schema,
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
	a, err := Parse(out.Message.Content)
	if err == nil && o.Style == StyleLabel {
		a.Quote = QuoteFor(paragraph, t)
	}
	return a, err
}
