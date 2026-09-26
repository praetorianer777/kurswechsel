// Package timeline orders a person's classified statements on a topic and
// marks changes of position.
package timeline

import (
	"sort"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/stance"
)

// Entry is one classified paragraph on a person's timeline.
type Entry struct {
	ParagraphID int64     `json:"paragraph_id"`
	SpeechID    string    `json:"speech_id"`
	Position    int       `json:"-"`
	Date        time.Time `json:"date"`
	Period      int       `json:"period"`
	Session     int       `json:"session"`
	Page        int       `json:"page,omitempty"`
	Faction     string    `json:"faction,omitempty"`
	Role        string    `json:"role,omitempty"`
	Text        string    `json:"text"`
	Stance      string    `json:"stance"`
	// Quote is empty when the model's quote is not verbatim in Text; the
	// website then shows the paragraph alone.
	Quote      string  `json:"quote,omitempty"`
	Rationale  string  `json:"rationale"`
	Confidence float64 `json:"confidence"`
	Model      string  `json:"model"`
	SourceURL  string  `json:"source_url"`
	// Change marks the first entry of a day whose position differs from the
	// previous day with a position; PreviousStance is that earlier position.
	Change         bool   `json:"change"`
	PreviousStance string `json:"previous_stance,omitempty"`
}

// Sort orders entries chronologically, keeping the order within a speech.
func Sort(es []Entry) {
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if !a.Date.Equal(b.Date) {
			return a.Date.Before(b.Date)
		}
		if a.SpeechID != b.SpeechID {
			return a.SpeechID < b.SpeechID
		}
		return a.Position < b.Position
	})
}

// MarkChanges sorts es and sets Change and PreviousStance.
//
// Positions are compared per day, not per paragraph: a speech often mixes a
// paraphrased opposing view with the speaker's own, and flagging that as a
// change within one speech would be wrong. A day's position is the majority of
// its "dafuer" and "dagegen" entries; a tie, or only neutral and unclear
// entries, gives the day no position, so it neither counts as a change nor
// resets the comparison.
func MarkChanges(es []Entry) int {
	Sort(es)
	changes := 0
	last := ""
	for start := 0; start < len(es); {
		end := start
		for end < len(es) && es[end].Date.Equal(es[start].Date) {
			end++
		}
		day := dayPosition(es[start:end])
		if day != "" {
			if last != "" && day != last {
				for i := start; i < end; i++ {
					if es[i].Stance == day {
						es[i].Change = true
						es[i].PreviousStance = last
						changes++
						break
					}
				}
			}
			last = day
		}
		start = end
	}
	return changes
}

func dayPosition(es []Entry) string {
	var pro, contra int
	for _, e := range es {
		switch e.Stance {
		case stance.For:
			pro++
		case stance.Against:
			contra++
		}
	}
	switch {
	case pro > contra:
		return stance.For
	case contra > pro:
		return stance.Against
	}
	return ""
}
