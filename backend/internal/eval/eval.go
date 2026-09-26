package eval

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// Report is the outcome of one evaluation run.
type Report struct {
	Classifier    string    `json:"classifier"`
	PromptVersion string    `json:"prompt_version"`
	Topic         string    `json:"topic"`
	Date          time.Time `json:"date"`
	// Items is how many gold items passed the keyword prefilter and were
	// sent to the classifier.
	Items     int           `json:"items"`
	Errors    int           `json:"errors"`
	Relevance Binary        `json:"relevance"`
	Stance    Confusion     `json:"stance"`
	Verbatim  int           `json:"verbatim_quotes"`
	Duration  time.Duration `json:"duration_ns"`
}

// Answered is one gold item with what the classifier said about it.
type Answered struct {
	Item   Item          `json:"item"`
	Answer stance.Answer `json:"answer"`
}

// Run classifies every gold item that passes the topic's keyword prefilter,
// the same path production data takes. Each answer is also passed to the
// optional observers, e.g. to dump them for later analysis.
func Run(ctx context.Context, c stance.Classifier, t topic.Topic, items []Item, now time.Time, observe ...func(Answered)) (Report, error) {
	r := Report{Classifier: c.Name(), PromptVersion: promptVersion(c), Topic: t.Slug, Date: now, Stance: Confusion{}}
	start := time.Now()
	for _, it := range items {
		if !t.Keywords.MatchString(it.Text) {
			// A relevant paragraph the prefilter drops is a miss of the
			// pipeline, so it counts against relevance recall.
			r.Relevance.Add(it.Relevant, false)
			continue
		}
		r.Items++
		a, err := c.Classify(ctx, t, it.Text)
		if err != nil {
			if ctx.Err() != nil {
				return r, ctx.Err()
			}
			r.Errors++
			continue
		}
		for _, o := range observe {
			o(Answered{Item: it, Answer: a})
		}
		r.Relevance.Add(it.Relevant, a.Relevant)
		if it.Relevant {
			got := a.Stance
			// Declaring a relevant paragraph irrelevant hides it from the
			// timeline, which is what "neutral" does too.
			if !a.Relevant {
				got = stance.Neutral
			}
			r.Stance.Add(it.Stance, got)
		}
		if a.Relevant && stance.Verbatim(it.Text, a.Quote) {
			r.Verbatim++
		}
	}
	r.Duration = time.Since(start)
	return r, nil
}

// promptVersion asks classifiers that support several prompts which one
// they use.
func promptVersion(c stance.Classifier) string {
	if p, ok := c.(interface{ Prompt() string }); ok {
		return p.Prompt()
	}
	return stance.PromptVersion
}

// WriteMarkdown prints the report for a terminal or an ADR.
func (r Report) WriteMarkdown(w io.Writer) error {
	predictedRelevant := r.Relevance.TP + r.Relevance.FP
	_, err := fmt.Fprintf(w, `Classifier: %s (prompt %s), topic %s, %d items, %d errors, %s

| Metric | Value |
| --- | ---: |
| Relevance precision | %.2f |
| Relevance recall | %.2f |
| Stance accuracy | %.2f |
| Stance macro-F1 | %.2f |
| For/against flipped | %.0f %% |
| „unklar“ rate | %.0f %% |
| Verbatim quotes | %d of %d |
`, r.Classifier, r.PromptVersion, r.Topic, r.Items, r.Errors, r.Duration.Round(time.Second),
		r.Relevance.Precision(), r.Relevance.Recall(), r.Stance.Accuracy(), r.Stance.MacroF1(),
		100*r.Stance.FlipRate(), 100*r.Stance.Rate(stance.Unclear), r.Verbatim, predictedRelevant)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "\n| gold \\ predicted |"); err != nil {
		return err
	}
	for _, l := range stance.Labels {
		fmt.Fprintf(w, " %s |", l)
	}
	fmt.Fprint(w, "\n| --- |")
	for range stance.Labels {
		fmt.Fprint(w, " ---: |")
	}
	fmt.Fprintln(w)
	for _, g := range stance.Labels {
		fmt.Fprintf(w, "| %s |", g)
		for _, p := range stance.Labels {
			fmt.Fprintf(w, " %d |", r.Stance[g][p])
		}
		fmt.Fprintln(w)
	}
	return nil
}
