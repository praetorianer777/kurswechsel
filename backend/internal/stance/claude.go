package stance

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// Claude classifies through the Anthropic API. It is optional: the project
// runs on local models by default, and nothing in the test suite calls the
// real API.
type Claude struct {
	Client anthropic.Client
	Model  string
}

// Name implements Classifier.
func (c *Claude) Name() string { return "claude/" + c.Model }

// Classify implements Classifier.
func (c *Claude) Classify(ctx context.Context, t topic.Topic, paragraph string) (Answer, error) {
	resp, err := c.Client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.Model,
		MaxTokens: 4096,
		System:    []anthropic.TextBlockParam{{Text: SystemPrompt(t)}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(UserPrompt(paragraph))),
		},
		OutputConfig: anthropic.OutputConfigParam{
			// A four-way label does not repay deep reasoning.
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: Schema()},
		},
	})
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			return Answer{}, fmt.Errorf("claude: HTTP %d: %w", apiErr.StatusCode, err)
		}
		return Answer{}, fmt.Errorf("claude: %w", err)
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return Answer{}, errors.New("claude: request refused")
	}
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			return Parse(tb.Text)
		}
	}
	return Answer{}, errors.New("claude: reply has no text block")
}
