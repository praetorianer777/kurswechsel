// Package stance is the spike's stance classifier: one fixed prompt, a JSON
// schema for the answer, and checks that the answer can be shown as evidence.
package stance

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/praetorianer777/kurswechsel/spikes/gold"
	"github.com/praetorianer777/kurswechsel/spikes/ollama"
)

// PromptVersion changes whenever System or the schema changes, so stored
// results can be traced to the prompt that produced them.
const PromptVersion = "wehrpflicht-v1"

// System is the instruction for every paragraph.
const System = `You classify paragraphs from speeches in the German Bundestag.

Topic: whether military service, or a general duty to serve that includes it, should be COMPULSORY in Germany (Wehrpflicht, verpflichtender Wehrdienst, Musterungspflicht, allgemeine Dienstpflicht).

Decide from the paragraph alone:
- relevant: true only if the paragraph substantively discusses that question, including the 2025 military service law. A passing mention (history, other countries, compensation for service-related injuries) is not relevant.
- stance (only meaningful when relevant):
  - "dafuer": the SPEAKER supports compulsion: reinstating conscription, compulsory screening of cohorts, a mandatory year of service, or criticises its suspension.
  - "dagegen": the SPEAKER opposes compulsion or insists on voluntary service only.
  - "neutral": facts, procedure or a model without taking a side on compulsion.
  - "unklar": irony, rhetorical questions, or other people's views; a position cannot be read off reliably.
- quote: the shortest passage that carries the decision, copied VERBATIM from the paragraph.
- rationale: one sentence in German explaining the decision, neutral in tone.
- confidence: 0 to 1.

Judge only the speaker's own position. Do not use your knowledge of the speaker or their party.`

// Schema constrains the model's reply.
var Schema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"relevant":   map[string]any{"type": "boolean"},
		"stance":     map[string]any{"type": "string", "enum": gold.Labels},
		"quote":      map[string]any{"type": "string"},
		"rationale":  map[string]any{"type": "string"},
		"confidence": map[string]any{"type": "number"},
	},
	"required": []string{"relevant", "stance", "quote", "rationale", "confidence"},
}

// Answer is the model's structured reply.
type Answer struct {
	Relevant   bool    `json:"relevant"`
	Stance     string  `json:"stance"`
	Quote      string  `json:"quote"`
	Rationale  string  `json:"rationale"`
	Confidence float64 `json:"confidence"`
}

// Result is one classified paragraph.
type Result struct {
	Answer
	// QuoteVerbatim reports whether the quote really occurs in the paragraph;
	// the website must never show words the speaker did not say.
	QuoteVerbatim bool
	Nanos         int64
	OutputTokens  int
}

// Chat is the part of the Ollama client the classifier needs.
type Chat interface {
	ChatJSON(ctx context.Context, model string, msgs []ollama.Message, schema any, think bool) (ollama.ChatResult, error)
}

// Model names an Ollama model and whether it runs with thinking on.
type Model struct {
	Name  string
	Think bool
}

// ParseModel reads "name" or "name@think".
func ParseModel(spec string) Model {
	name, think := strings.CutSuffix(spec, "@think")
	return Model{Name: name, Think: think}
}

func (m Model) String() string {
	if m.Think {
		return m.Name + " (thinking)"
	}
	return m.Name
}

// Classify asks model about one paragraph.
func Classify(ctx context.Context, c Chat, model Model, it gold.Item) (Result, error) {
	msgs := []ollama.Message{
		{Role: "system", Content: System},
		{Role: "user", Content: fmt.Sprintf("Paragraph:\n%s", it.Text)},
	}
	res, err := c.ChatJSON(ctx, model.Name, msgs, Schema, model.Think)
	if err != nil {
		return Result{}, err
	}
	a, err := Parse(res.Content)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Answer:        a,
		QuoteVerbatim: Verbatim(it.Text, a.Quote),
		Nanos:         res.TotalNanos,
		OutputTokens:  res.OutputTokens,
	}, nil
}

// Parse decodes and validates a reply. Models that ignore the schema tend to
// wrap the object in prose or a code fence, so the outermost braces are tried
// before giving up.
func Parse(content string) (Answer, error) {
	var a Answer
	if err := json.Unmarshal([]byte(content), &a); err != nil {
		i, j := strings.Index(content, "{"), strings.LastIndex(content, "}")
		if i < 0 || j < i || json.Unmarshal([]byte(content[i:j+1]), &a) != nil {
			return a, fmt.Errorf("reply is not JSON: %w", err)
		}
	}
	if !slices.Contains(gold.Labels, a.Stance) {
		return a, fmt.Errorf("unknown stance %q", a.Stance)
	}
	return a, nil
}

// Verbatim reports whether quote occurs in text, ignoring whitespace
// differences and surrounding quotation marks.
func Verbatim(text, quote string) bool {
	q := strings.Join(strings.Fields(strings.Trim(quote, " \"'„“”»«")), " ")
	if q == "" {
		return false
	}
	return strings.Contains(strings.Join(strings.Fields(text), " "), q)
}
