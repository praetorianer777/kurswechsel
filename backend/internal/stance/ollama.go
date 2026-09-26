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
	// Style is StyleFull (the default when empty), StyleLabel or StyleProbs.
	Style string
	// MinConfidence applies to StyleProbs: a side below it becomes "unklar".
	MinConfidence float64
	HTTP          *http.Client
}

// Name implements Classifier. The label-only style is part of the name so
// its stored results are never mistaken for full-prompt ones.
func (o *Ollama) Name() string {
	switch o.Style {
	case StyleLabel:
		return "ollama/" + o.Model + "+label"
	case StyleProbs:
		if o.MinConfidence > 0 {
			return fmt.Sprintf("ollama/%s+probs@%.2f", o.Model, o.MinConfidence)
		}
		return "ollama/" + o.Model + "+probs"
	}
	return "ollama/" + o.Model
}

// Prompt reports the prompt version this classifier uses.
func (o *Ollama) Prompt() string {
	switch o.Style {
	case StyleLabel:
		return LabelPromptVersion
	case StyleProbs:
		return ProbsPromptVersion
	}
	return PromptVersion
}

// Classify implements Classifier.
func (o *Ollama) Classify(ctx context.Context, t topic.Topic, paragraph string) (Answer, error) {
	options := map[string]any{"temperature": 0, "seed": 1, "num_ctx": 8192}
	req := map[string]any{"model": o.Model, "stream": false, "options": options}
	system := SystemPrompt(t)
	switch o.Style {
	case StyleLabel:
		system = LabelSystemPrompt(t)
		req["format"] = LabelSchema()
	case StyleProbs:
		// No schema: it would make the first token "{" instead of the letter.
		system = ProbsSystemPrompt(t)
		options["num_predict"] = 1
		req["logprobs"] = true
		req["top_logprobs"] = 20
	default:
		req["format"] = Schema()
	}
	req["messages"] = []map[string]string{
		{"role": "system", "content": system},
		{"role": "user", "content": UserPrompt(paragraph)},
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
		Logprobs []struct {
			TopLogprobs []logprob `json:"top_logprobs"`
		} `json:"logprobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Answer{}, fmt.Errorf("ollama: %w", err)
	}
	if o.Style == StyleProbs {
		if len(out.Logprobs) == 0 {
			return Answer{}, fmt.Errorf("ollama: no logprobs in reply; needs Ollama with logprobs support")
		}
		probs, err := letterProbs(out.Logprobs[0].TopLogprobs)
		if err != nil {
			return Answer{}, fmt.Errorf("ollama: %w (reply %q)", err, out.Message.Content)
		}
		return probsAnswer(probs, o.MinConfidence, paragraph, t), nil
	}
	a, err := Parse(out.Message.Content)
	if err == nil && o.Style == StyleLabel {
		a.Quote = QuoteFor(paragraph, t)
	}
	return a, err
}
