package timeline

import (
	"testing"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
)

const (
	F = stance.For
	A = stance.Against
	N = stance.Neutral
	U = stance.Unclear
)

// e builds an entry on day d (days since the epoch) in speech s.
func e(d int, s string, pos int, st string) Entry {
	return Entry{Date: time.Unix(int64(d)*86400, 0).UTC(), SpeechID: s, Position: pos, Stance: st}
}

func changes(es []Entry) []string {
	var out []string
	for _, x := range es {
		if x.Change {
			out = append(out, x.PreviousStance+"→"+x.Stance+"@"+x.Date.Format("01-02"))
		}
	}
	return out
}

func TestMarkChanges(t *testing.T) {
	tests := []struct {
		name string
		in   []Entry
		want []string
	}{
		{name: "empty", in: nil},
		{name: "single entry", in: []Entry{e(1, "a", 0, F)}},
		{name: "consistent", in: []Entry{e(1, "a", 0, F), e(2, "b", 0, F), e(3, "c", 0, F)}},
		{name: "one change", in: []Entry{e(1, "a", 0, F), e(5, "b", 0, A)}, want: []string{"dafuer→dagegen@01-06"}},
		{name: "change and back", in: []Entry{e(1, "a", 0, A), e(2, "b", 0, F), e(3, "c", 0, A)},
			want: []string{"dagegen→dafuer@01-03", "dafuer→dagegen@01-04"}},
		{name: "neutral and unclear in between do not reset", in: []Entry{e(1, "a", 0, F), e(2, "b", 0, N), e(3, "c", 0, U), e(4, "d", 0, F)}},
		{name: "change across neutral days", in: []Entry{e(1, "a", 0, F), e(2, "b", 0, N), e(3, "c", 0, A)}, want: []string{"dafuer→dagegen@01-04"}},
		{name: "mixed speech same day is not a change", in: []Entry{e(1, "a", 0, A), e(1, "a", 1, F), e(1, "a", 2, A)}},
		{name: "day majority decides", in: []Entry{e(1, "a", 0, F), e(2, "b", 0, F), e(2, "b", 1, A), e(2, "b", 2, A)},
			want: []string{"dafuer→dagegen@01-03"}},
		{name: "tie gives no position", in: []Entry{e(1, "a", 0, F), e(2, "b", 0, F), e(2, "b", 1, A), e(3, "c", 0, F)}},
		{name: "unsorted input", in: []Entry{e(5, "b", 0, A), e(1, "a", 0, F)}, want: []string{"dafuer→dagegen@01-06"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := MarkChanges(tt.in)
			got := changes(tt.in)
			if n != len(tt.want) || len(got) != len(tt.want) {
				t.Fatalf("changes = %v (n=%d), want %v", got, n, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("change %d = %s, want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSortKeepsSpeechOrder(t *testing.T) {
	es := []Entry{e(2, "b", 0, N), e(1, "a", 2, N), e(1, "a", 0, N), e(1, "0", 5, N)}
	Sort(es)
	got := ""
	for _, x := range es {
		got += x.SpeechID + string(rune('0'+x.Position))
	}
	if got != "05a0a2b0" {
		t.Errorf("order = %s", got)
	}
}
