// Package classify runs a stance classifier over a topic's candidates.
package classify

import (
	"context"
	"fmt"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// MaxConsecutiveFailures stops a run when the classifier is clearly down
// instead of logging the same error for every remaining paragraph.
const MaxConsecutiveFailures = 5

// Options controls a run.
type Options struct {
	Limit int
	// Redo discards earlier classifications of the topic first.
	Redo bool
	Logf func(format string, args ...any)
	Now  func() time.Time
}

// Summary reports what a run did.
type Summary struct {
	Candidates int
	Classified int
	Relevant   int
	Failed     int
}

// Run assigns candidates by keyword, then classifies those still pending.
func Run(ctx context.Context, st *store.Store, c stance.Classifier, t topic.Topic, o Options) (Summary, error) {
	var sum Summary
	logf, now := o.Logf, o.Now
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if now == nil {
		now = time.Now
	}
	if err := st.SyncTopics(ctx, topic.All); err != nil {
		return sum, err
	}
	n, err := st.AssignTopic(ctx, t)
	if err != nil {
		return sum, err
	}
	sum.Candidates = n
	if o.Redo {
		if err := st.DeleteStances(ctx, t.Slug); err != nil {
			return sum, err
		}
	}
	pending, err := st.Pending(ctx, t.Slug, o.Limit)
	if err != nil {
		return sum, err
	}
	logf("%s: %d candidates, %d to classify with %s", t.Slug, n, len(pending), c.Name())

	failures := 0
	for i, cand := range pending {
		a, err := c.Classify(ctx, t, cand.Text)
		if err != nil {
			if ctx.Err() != nil {
				return sum, ctx.Err()
			}
			sum.Failed++
			failures++
			logf("paragraph %d: %v", cand.ParagraphID, err)
			if failures >= MaxConsecutiveFailures {
				return sum, fmt.Errorf("%d classifications in a row failed, last: %w", failures, err)
			}
			continue
		}
		failures = 0
		if err := st.SaveStance(ctx, cand.ParagraphID, t.Slug, a, stance.Verbatim(cand.Text, a.Quote), c.Name(), now()); err != nil {
			return sum, err
		}
		sum.Classified++
		if a.Relevant {
			sum.Relevant++
		}
		if (i+1)%25 == 0 {
			logf("%d/%d classified", i+1, len(pending))
		}
	}
	return sum, nil
}
