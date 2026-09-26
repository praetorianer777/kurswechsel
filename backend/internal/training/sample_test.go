package training

import (
	"fmt"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/store"
)

func cands() []store.CandidateInfo {
	var out []store.CandidateInfo
	for i := range 30 {
		faction := "A"
		if i%3 == 0 {
			faction = "B"
		}
		out = append(out, store.CandidateInfo{ID: fmt.Sprintf("ID%02d#0", i), Period: 19 + i%2, Faction: faction})
	}
	return out
}

func TestSample(t *testing.T) {
	excl := map[string]bool{"ID00#0": true, "ID01#0": true}
	a := Sample(cands(), 8, excl, 1)
	b := Sample(cands(), 8, excl, 1)
	if len(a) != 8 || fmt.Sprint(a) != fmt.Sprint(b) {
		t.Fatalf("not reproducible or wrong size: %d", len(a))
	}
	groups := map[string]int{}
	for _, c := range a {
		if excl[c.ID] {
			t.Errorf("excluded %s sampled", c.ID)
		}
		groups[fmt.Sprint(c.Period, c.Faction)]++
	}
	if len(groups) != 4 {
		t.Errorf("not spread over all period/faction groups: %v", groups)
	}
	if got := Sample(cands(), 100, nil, 1); len(got) != 30 {
		t.Errorf("asking for more than exist returns %d, want all 30", len(got))
	}
}
