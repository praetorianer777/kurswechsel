// Package stance classifies what a speaker's paragraph says about a topic.
package stance

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// Labels.
const (
	For     = "dafuer"
	Against = "dagegen"
	Neutral = "neutral"
	Unclear = "unklar"
)

// Labels lists every stance label.
var Labels = []string{For, Against, Neutral, Unclear}

// PromptVersion changes whenever SystemPrompt or Schema change meaning, so
// stored results show which prompt produced them and can be re-run.
const PromptVersion = "v1"

// Answer is what a classifier says about one paragraph.
type Answer struct {
	Relevant   bool    `json:"relevant"`
	Stance     string  `json:"stance"`
	Quote      string  `json:"quote"`
	Rationale  string  `json:"rationale"`
	Confidence float64 `json:"confidence"`
}

// Classifier judges one paragraph.
type Classifier interface {
	// Name identifies provider and model, e.g. "ollama/qwen3:30b-a3b".
	Name() string
	Classify(ctx context.Context, t topic.Topic, paragraph string) (Answer, error)
}

// SystemPrompt is the instruction for every paragraph on topic t.
func SystemPrompt(t topic.Topic) string {
	return `You classify paragraphs from speeches in the German Bundestag.

` + t.Definition + `

Also give:
- quote: the shortest passage that carries the decision, copied VERBATIM from the paragraph.
- rationale: one sentence in German explaining the decision, neutral in tone.
- confidence: 0 to 1.

Decide from the paragraph alone. Judge only the speaker's own position; do not use your knowledge of the speaker or their party.`
}

// UserPrompt wraps the paragraph.
func UserPrompt(paragraph string) string { return "Paragraph:\n" + paragraph }

// Schema constrains the reply. additionalProperties is false because
// structured-output implementations require closed objects.
func Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"relevant":   map[string]any{"type": "boolean"},
			"stance":     map[string]any{"type": "string", "enum": Labels},
			"quote":      map[string]any{"type": "string"},
			"rationale":  map[string]any{"type": "string"},
			"confidence": map[string]any{"type": "number"},
		},
		"required":             []string{"relevant", "stance", "quote", "rationale", "confidence"},
		"additionalProperties": false,
	}
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
	if !slices.Contains(Labels, a.Stance) {
		return a, fmt.Errorf("unknown stance %q", a.Stance)
	}
	a.Confidence = min(max(a.Confidence, 0), 1)
	return a, nil
}

// Verbatim reports whether quote occurs in text, ignoring whitespace
// differences and surrounding quotation marks. The website shows a quote only
// when this holds, so it never puts words in a speaker's mouth.
func Verbatim(text, quote string) bool {
	q := strings.Join(strings.Fields(strings.Trim(quote, " \"'„“”»«")), " ")
	if q == "" {
		return false
	}
	return strings.Contains(strings.Join(strings.Fields(text), " "), q)
}
