package stance

import (
	"context"
	"strings"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// ContextPromptVersion identifies prompt v1 with surrounding paragraphs.
const ContextPromptVersion = "v1-context"

// Context is what surrounds a paragraph. It helps to read the paragraph
// (what it answers, what "das" refers to) but is never classified itself.
type Context struct {
	PreviousSpeaker string
	Previous        string
	Before          []string
	After           []string
}

// Empty reports whether there is no context at all.
func (c Context) Empty() bool {
	return c.Previous == "" && len(c.Before) == 0 && len(c.After) == 0
}

// ContextClassifier can take the surrounding paragraphs into account.
type ContextClassifier interface {
	Classifier
	ClassifyInContext(ctx context.Context, t topic.Topic, paragraph string, c Context) (Answer, error)
}

const contextRule = `

The paragraph may come with context from the same debate: the end of the previous speaker's speech and neighbouring paragraphs of the same speech. Use the context only to understand the paragraph, for example what it replies to or what "das" refers to. Classify only the paragraph under "Paragraph to classify", and take the quote only from it.`

// ContextSystemPrompt is SystemPrompt with the rule for using context.
func ContextSystemPrompt(t topic.Topic) string { return SystemPrompt(t) + contextRule }

// ContextUserPrompt puts the context around the paragraph, clearly marked.
func ContextUserPrompt(paragraph string, c Context) string {
	var b strings.Builder
	if c.Previous != "" {
		b.WriteString("Context – end of the previous speech")
		if c.PreviousSpeaker != "" {
			b.WriteString(" (" + c.PreviousSpeaker + ")")
		}
		b.WriteString(":\n" + c.Previous + "\n\n")
	}
	if len(c.Before) > 0 {
		b.WriteString("Context – earlier in the same speech:\n" + strings.Join(c.Before, "\n\n") + "\n\n")
	}
	b.WriteString("Paragraph to classify:\n" + paragraph)
	if len(c.After) > 0 {
		b.WriteString("\n\nContext – later in the same speech:\n" + strings.Join(c.After, "\n\n"))
	}
	return b.String()
}
