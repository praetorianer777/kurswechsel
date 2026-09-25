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

// Run classifies every gold item that passes the topic's keyword prefilter,
// the same path production data takes.
func Run(ctx context.Context, c stance.Classifier, t topic.Topic, items []Item, now time.Time) (Report, error) {
	r := Report{Classifier: c.Name(), PromptVersion: stance.PromptVersion, Topic: t.Slug, Date: now, Stance: Confusion{}}
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
