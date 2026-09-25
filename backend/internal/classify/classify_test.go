package classify

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

var ctx = context.Background()

func seeded(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	p := bundestag.Protocol{
		Period: 20, Session: 1, Date: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Speeches: []bundestag.Speech{{
			ID:      "ID1",
			Speaker: bundestag.Speaker{ID: "1", First: "A", Last: "B", Faction: "SPD"},
			Paragraphs: []bundestag.Paragraph{
				{Text: "Wir müssen die Wehrpflicht wieder einführen."},
				{Text: "Die Bundeswehr braucht Panzer."},
				{Text: "Der Wehrdienst muss freiwillig bleiben."},
				{Text: "„Die Wehrpflicht ist ein Relikt“, schrieb die Zeitung.", Quote: true},
			},
		}},
	}
	if err := st.SaveProtocol(ctx, p, "u", time.Now()); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestRun(t *testing.T) {
	st := seeded(t)
	sum, err := Run(ctx, st, stance.Fake{}, topic.Wehrpflicht, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{Candidates: 2, Classified: 2, Relevant: 2}) {
		t.Errorf("summary = %+v (the quotation must not be a candidate)", sum)
	}

	var stances []string
	rows, _ := st.DB().Query(`SELECT stance || '/' || quote_verbatim || '/' || model FROM stances ORDER BY paragraph_id`)
	for rows.Next() {
		var s string
		rows.Scan(&s)
		stances = append(stances, s)
	}
	rows.Close()
	if len(stances) != 2 || stances[0] != "dafuer/1/fake" || stances[1] != "dagegen/1/fake" {
		t.Errorf("stances = %v", stances)
	}

	sum, err = Run(ctx, st, stance.Fake{}, topic.Wehrpflicht, Options{})
	if err != nil || sum.Classified != 0 {
		t.Errorf("second run = %+v, %v; want nothing left to classify", sum, err)
	}
	sum, err = Run(ctx, st, stance.Fake{}, topic.Wehrpflicht, Options{Redo: true, Limit: 1})
	if err != nil || sum.Classified != 1 {
		t.Errorf("redo with limit = %+v, %v", sum, err)
	}
}

type failing struct{ err error }

func (failing) Name() string { return "failing" }
func (f failing) Classify(context.Context, topic.Topic, string) (stance.Answer, error) {
	return stance.Answer{}, f.err
}

func TestRunStopsWhenClassifierIsDown(t *testing.T) {
	st := seeded(t)
	for i := range 6 {
		p := bundestag.Protocol{
			Period: 20, Session: 2 + i, Date: time.Date(2024, 2, 1+i, 0, 0, 0, 0, time.UTC),
			Speeches: []bundestag.Speech{{
				ID:         "IDX" + string(rune('a'+i)),
				Speaker:    bundestag.Speaker{ID: "1", First: "A", Last: "B"},
				Paragraphs: []bundestag.Paragraph{{Text: "Die Musterung kommt."}},
			}},
		}
		if err := st.SaveProtocol(ctx, p, "u", time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	var logged int
	sum, err := Run(ctx, st, failing{errors.New("connection refused")}, topic.Wehrpflicht,
		Options{Logf: func(string, ...any) { logged++ }})
	if err == nil {
		t.Fatal("want error after repeated failures")
	}
	if sum.Failed != MaxConsecutiveFailures || logged == 0 {
		t.Errorf("summary = %+v, logged %d", sum, logged)
	}
}

func TestRunCancelled(t *testing.T) {
	st := seeded(t)
	c, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Run(c, st, failing{context.Canceled}, topic.Wehrpflicht, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
