package eval

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func TestReadGold(t *testing.T) {
	in := `{"id":"1","text":"a","relevant":true,"stance":"dafuer"}

{"id":"2","text":"b","relevant":false}
`
	items, err := ReadGold(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Stance != stance.For || items[1].Relevant {
		t.Errorf("got %+v", items)
	}
}

func TestReadGoldRejects(t *testing.T) {
	for name, in := range map[string]string{
		"missing text":         `{"id":"1","relevant":false}`,
		"unknown stance":       `{"id":"1","text":"a","relevant":true,"stance":"ja"}`,
		"stance on irrelevant": `{"id":"1","text":"a","relevant":false,"stance":"neutral"}`,
		"relevant w/o stance":  `{"id":"1","text":"a","relevant":true}`,
		"broken json":          `{"id":`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadGold(strings.NewReader(in)); err == nil {
				t.Error("want error")
			}
		})
	}
}

func TestBinary(t *testing.T) {
	var b Binary
	for _, c := range [][2]bool{{true, true}, {true, true}, {false, true}, {true, false}, {false, false}} {
		b.Add(c[0], c[1])
	}
	if b.Precision() != 2.0/3 || b.Recall() != 2.0/3 || math.Abs(b.F1()-2.0/3) > 1e-9 {
		t.Errorf("P=%v R=%v F1=%v", b.Precision(), b.Recall(), b.F1())
	}
	if (Binary{}).F1() != 0 {
		t.Error("empty F1 must be 0, not NaN")
	}
}

func TestConfusion(t *testing.T) {
	c := Confusion{}
	c.Add(stance.For, stance.For)
	c.Add(stance.For, stance.Against)
	c.Add(stance.Against, stance.Against)
	c.Add(stance.Neutral, stance.Unclear)

	if c.Accuracy() != 0.5 || c.Rate(stance.Unclear) != 0.25 || c.Total() != 4 {
		t.Errorf("accuracy %v, unclear rate %v", c.Accuracy(), c.Rate(stance.Unclear))
	}
	if got, want := c.MacroF1(), (2.0/3+2.0/3)/4; math.Abs(got-want) > 1e-9 {
		t.Errorf("MacroF1 = %v, want %v", got, want)
	}
	if got := c.FlipRate(); math.Abs(got-1.0/3) > 1e-9 {
		t.Errorf("FlipRate = %v, want 1/3", got)
	}
	if (Confusion{}).MacroF1() != 0 || (Confusion{}).FlipRate() != 0 {
		t.Error("empty confusion must score 0")
	}
}

func goldSet() []Item {
	return []Item{
		{ID: "1", Text: "Wir müssen die Wehrpflicht wieder einführen.", Relevant: true, Stance: stance.For},
		{ID: "2", Text: "Der Wehrdienst muss freiwillig bleiben.", Relevant: true, Stance: stance.Against},
		{ID: "3", Text: "Die Musterung ist Thema im Ausschuss.", Relevant: false},
		{ID: "4", Text: "Die Bundeswehr braucht Panzer.", Relevant: false},
		{ID: "5", Text: "Niemand soll gezwungen werden zu dienen.", Relevant: true, Stance: stance.Against},
	}
}

func TestRun(t *testing.T) {
	r, err := Run(context.Background(), stance.Fake{}, topic.Wehrpflicht, goldSet(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if r.Items != 3 {
		t.Errorf("items = %d, want 3 through the prefilter", r.Items)
	}
	// Item 5 is relevant but has no keyword: a pipeline miss.
	if r.Relevance.FN != 1 || r.Relevance.TP != 2 || r.Relevance.FP != 1 {
		t.Errorf("relevance = %+v", r.Relevance)
	}
	if r.Stance.Accuracy() != 1 || r.Verbatim != 3 {
		t.Errorf("stance accuracy %v, verbatim %d", r.Stance.Accuracy(), r.Verbatim)
	}

	var buf bytes.Buffer
	if err := r.WriteMarkdown(&buf); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Classifier: fake (prompt v1)", "| Stance accuracy | 1.00 |", "| dafuer | 1 | 0 | 0 | 0 |"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("markdown lacks %q:\n%s", want, buf.String())
		}
	}
}

type broken struct{}

func (broken) Name() string { return "broken" }
func (broken) Classify(context.Context, topic.Topic, string) (stance.Answer, error) {
	return stance.Answer{}, errors.New("boom")
}

func TestRunCountsErrors(t *testing.T) {
	r, err := Run(context.Background(), broken{}, topic.Wehrpflicht, goldSet(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if r.Errors != 3 || r.Stance.Total() != 0 {
		t.Errorf("report = %+v", r)
	}
}
