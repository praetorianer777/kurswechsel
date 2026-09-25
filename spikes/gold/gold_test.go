package gold

import (
	"math"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	in := `{"id":"1","text":"a","relevant":true,"stance":"dafuer"}

{"id":"2","text":"b","relevant":false}
`
	items, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Stance != For || items[1].Relevant {
		t.Errorf("got %+v", items)
	}
}

func TestReadRejects(t *testing.T) {
	for name, in := range map[string]string{
		"missing text":         `{"id":"1","relevant":false}`,
		"unknown stance":       `{"id":"1","text":"a","relevant":true,"stance":"ja"}`,
		"stance on irrelevant": `{"id":"1","text":"a","relevant":false,"stance":"neutral"}`,
		"relevant w/o stance":  `{"id":"1","text":"a","relevant":true}`,
		"broken json":          `{"id":`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Read(strings.NewReader(in)); err == nil {
				t.Error("want error")
			}
		})
	}
}

func TestBinary(t *testing.T) {
	var b Binary
	b.Add(true, true)
	b.Add(true, true)
	b.Add(false, true)
	b.Add(true, false)
	b.Add(false, false)
	if b.Precision() != 2.0/3 || b.Recall() != 2.0/3 || math.Abs(b.F1()-2.0/3) > 1e-9 {
		t.Errorf("P=%v R=%v F1=%v", b.Precision(), b.Recall(), b.F1())
	}
	if (Binary{}).Precision() != 0 {
		t.Error("empty precision must be 0, not NaN")
	}
}

func TestConfusion(t *testing.T) {
	c := Confusion{}
	c.Add(For, For)
	c.Add(For, Against)
	c.Add(Against, Against)
	c.Add(Neutral, Unclear)

	if got := c.Accuracy(); got != 0.5 {
		t.Errorf("Accuracy = %v, want 0.5", got)
	}
	if got := c.Rate(Unclear); got != 0.25 {
		t.Errorf("Rate(unklar) = %v, want 0.25", got)
	}
	// for: P=1 R=.5 F1=2/3; against: P=.5 R=1 F1=2/3; neutral: 0; unclear: 0.
	if got, want := c.MacroF1(), (2.0/3+2.0/3)/4; math.Abs(got-want) > 1e-9 {
		t.Errorf("MacroF1 = %v, want %v", got, want)
	}
	if (Confusion{}).MacroF1() != 0 {
		t.Error("empty MacroF1 must be 0")
	}
}
