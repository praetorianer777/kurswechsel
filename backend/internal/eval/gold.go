// Package eval measures a stance classifier against a hand-labelled gold set.
package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/praetorianer777/kurswechsel/internal/stance"
)

// Item is one labelled paragraph. Stance is empty for paragraphs that are not
// about the topic.
type Item struct {
	ID       string `json:"id"`
	SpeechID string `json:"speech_id"`
	Date     string `json:"date"`
	Speaker  string `json:"speaker"`
	Party    string `json:"party"`
	Text     string `json:"text"`
	Relevant bool   `json:"relevant"`
	Stance   string `json:"stance,omitempty"`
	Note     string `json:"note,omitempty"`
}

// ReadGold parses JSON Lines and validates every label.
func ReadGold(r io.Reader) ([]Item, error) {
	var items []Item
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for line := 1; sc.Scan(); line++ {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var it Item
		if err := json.Unmarshal(sc.Bytes(), &it); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		switch {
		case it.ID == "" || it.Text == "":
			return nil, fmt.Errorf("line %d: id and text are required", line)
		case it.Relevant && !slices.Contains(stance.Labels, it.Stance):
			return nil, fmt.Errorf("line %d: relevant item needs a stance, got %q", line, it.Stance)
		case !it.Relevant && it.Stance != "":
			return nil, fmt.Errorf("line %d: irrelevant item must not have a stance", line)
		}
		items = append(items, it)
	}
	return items, sc.Err()
}

// Binary counts a yes/no decision.
type Binary struct{ TP, FP, FN, TN int }

// Add records one decision.
func (b *Binary) Add(want, got bool) {
	switch {
	case want && got:
		b.TP++
	case !want && got:
		b.FP++
	case want && !got:
		b.FN++
	default:
		b.TN++
	}
}

func (b Binary) Precision() float64 { return ratio(b.TP, b.TP+b.FP) }
func (b Binary) Recall() float64    { return ratio(b.TP, b.TP+b.FN) }
func (b Binary) F1() float64        { return f1(b.Precision(), b.Recall()) }

// Confusion counts gold label × predicted label.
type Confusion map[string]map[string]int

// Add records one prediction.
func (c Confusion) Add(want, got string) {
	if c[want] == nil {
		c[want] = map[string]int{}
	}
	c[want][got]++
}

// Total is the number of predictions.
func (c Confusion) Total() int {
	n := 0
	for _, row := range c {
		for _, v := range row {
			n += v
		}
	}
	return n
}

// Accuracy is the share of exact matches.
func (c Confusion) Accuracy() float64 {
	hit := 0
	for _, l := range stance.Labels {
		hit += c[l][l]
	}
	return ratio(hit, c.Total())
}

// MacroF1 averages per-label F1 over the labels that occur in gold or
// predictions, so an unused label does not count as a perfect score.
func (c Confusion) MacroF1() float64 {
	var sum float64
	n := 0
	for _, l := range stance.Labels {
		tp, fp, fn := c[l][l], 0, 0
		for _, o := range stance.Labels {
			if o != l {
				fp += c[o][l]
				fn += c[l][o]
			}
		}
		if tp+fp+fn == 0 {
			continue
		}
		sum += f1(ratio(tp, tp+fp), ratio(tp, tp+fn))
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// Rate is the share of predictions equal to label.
func (c Confusion) Rate(label string) float64 {
	n := 0
	for _, row := range c {
		n += row[label]
	}
	return ratio(n, c.Total())
}

// FlipRate is the share of gold "for" or "against" items predicted as the
// opposite side: the error that would draw a false change of position.
func (c Confusion) FlipRate() float64 {
	flips := c[stance.For][stance.Against] + c[stance.Against][stance.For]
	sided := 0
	for _, l := range []string{stance.For, stance.Against} {
		for _, v := range c[l] {
			sided += v
		}
	}
	return ratio(flips, sided)
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func f1(p, r float64) float64 {
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}
