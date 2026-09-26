package training

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
)

// Item is one paragraph of the training set with its pre-label and, once
// checked, the reviewer's verdict.
type Item struct {
	ID       string `json:"id"`
	SpeechID string `json:"speech_id"`
	Date     string `json:"date"`
	Speaker  string `json:"speaker"`
	Party    string `json:"party"`
	Text     string `json:"text"`

	Relevant  bool   `json:"relevant"`
	Stance    string `json:"stance,omitempty"`
	Quote     string `json:"quote,omitempty"`
	Note      string `json:"note,omitempty"`
	Unsure    bool   `json:"unsure,omitempty"`
	LabeledBy string `json:"labeled_by,omitempty"`

	// Sampled marks a sure pre-label drawn for the agreement check.
	Sampled bool    `json:"sampled,omitempty"`
	Review  *Review `json:"review,omitempty"`
}

// Review is a person's verdict on a pre-label.
type Review struct {
	Relevant bool      `json:"relevant"`
	Stance   string    `json:"stance,omitempty"`
	Note     string    `json:"note,omitempty"`
	At       time.Time `json:"at"`
}

// Labelled reports whether the item carries a pre-label.
func (it Item) Labelled() bool { return it.LabeledBy != "" }

// NeedsReview reports whether a person has to look at the item.
func (it Item) NeedsReview() bool { return it.Labelled() && (it.Unsure || it.Sampled) }

// Agrees reports whether the review confirmed the pre-label.
func (it Item) Agrees() bool {
	if it.Review == nil {
		return false
	}
	if it.Review.Relevant != it.Relevant {
		return false
	}
	return !it.Relevant || it.Review.Stance == it.Stance
}

// Final is the label that goes into training: the review if there is one,
// otherwise the pre-label.
func (it Item) Final() (relevant bool, st string) {
	if it.Review != nil {
		return it.Review.Relevant, it.Review.Stance
	}
	return it.Relevant, it.Stance
}

// Validate checks a label for consistency.
func Validate(relevant bool, st string) error {
	if relevant && !slices.Contains(stance.Labels, st) {
		return fmt.Errorf("relevant item needs a stance, got %q", st)
	}
	if !relevant && st != "" {
		return fmt.Errorf("irrelevant item must not have a stance")
	}
	return nil
}

// Read loads a training file.
func Read(path string) ([]Item, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var items []Item
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for line := 1; sc.Scan(); line++ {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var it Item
		if err := json.Unmarshal(sc.Bytes(), &it); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if it.Labelled() {
			if err := Validate(it.Relevant, it.Stance); err != nil {
				return nil, fmt.Errorf("%s:%d: %w", path, line, err)
			}
		}
		items = append(items, it)
	}
	return items, sc.Err()
}

// Write saves a training file atomically, so a crash mid-review never
// leaves a half-written file behind.
func Write(path string, items []Item) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".training-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	for _, it := range items {
		if err := enc.Encode(it); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// DrawSample marks n of the sure pre-labels for the agreement check, unless
// a sample was drawn already. The draw is reproducible.
func DrawSample(items []Item, n int, seed uint64) {
	var sure []int
	for i, it := range items {
		if it.Sampled {
			return
		}
		if it.Labelled() && !it.Unsure {
			sure = append(sure, i)
		}
	}
	r := rand.New(rand.NewPCG(seed, 50))
	r.Shuffle(len(sure), func(i, j int) { sure[i], sure[j] = sure[j], sure[i] })
	for _, i := range sure[:min(n, len(sure))] {
		items[i].Sampled = true
	}
}

// Stats summarises review progress.
type Stats struct {
	Items, Labelled, Unsure, Sampled int
	ToReview, Reviewed               int
	SampleReviewed, SampleAgreed     int
}

// Agreement is the share of reviewed sample items whose pre-label was
// confirmed, the measure that decides whether the rest can be accepted.
func (s Stats) Agreement() float64 {
	if s.SampleReviewed == 0 {
		return 0
	}
	return float64(s.SampleAgreed) / float64(s.SampleReviewed)
}

// Summarise counts the items.
func Summarise(items []Item) Stats {
	var s Stats
	for _, it := range items {
		s.Items++
		if !it.Labelled() {
			continue
		}
		s.Labelled++
		if it.Unsure {
			s.Unsure++
		}
		if it.Sampled {
			s.Sampled++
		}
		if it.NeedsReview() {
			s.ToReview++
			if it.Review != nil {
				s.Reviewed++
			}
		}
		if it.Sampled && it.Review != nil {
			s.SampleReviewed++
			if it.Agrees() {
				s.SampleAgreed++
			}
		}
	}
	return s
}
