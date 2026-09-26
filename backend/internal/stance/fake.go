package stance

import (
	"context"
	"strings"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// Fake is a deterministic rule-based classifier for tests and demo data. Its
// answers are plausible, not correct.
type Fake struct{}

// Name implements Classifier.
func (Fake) Name() string { return "fake" }

// Classify implements Classifier.
func (Fake) Classify(_ context.Context, t topic.Topic, paragraph string) (Answer, error) {
	quote := firstSentenceMatching(paragraph, t)
	if quote == "" {
		return Answer{Stance: Neutral, Quote: "", Rationale: "Der Absatz behandelt das Thema nicht.", Confidence: 0.9}, nil
	}
	lower := strings.ToLower(paragraph)
	a := Answer{Relevant: true, Stance: Neutral, Quote: quote, Rationale: "Der Absatz beschreibt das Thema ohne eigene Position.", Confidence: 0.6}
	switch {
	case containsAny(lower, "freiwillig", "keine zwang", "kein zwang", "keinen zwang", "ablehnen", "lehnen"):
		a.Stance, a.Rationale = Against, "Der Redner spricht sich gegen eine Pflicht aus."
	case containsAny(lower, "wieder einführen", "wiedereinführung", "brauchen die wehrpflicht", "verpflichtend", "pflicht für alle"):
		a.Stance, a.Rationale = For, "Der Redner spricht sich für eine Pflicht aus."
	case strings.Contains(paragraph, "?"):
		a.Stance, a.Rationale = Unclear, "Der Absatz stellt eine Frage, ohne Position zu beziehen."
	}
	return a, nil
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func firstSentenceMatching(paragraph string, t topic.Topic) string {
	rest := paragraph
	for rest != "" {
		end := strings.IndexAny(rest, ".!?")
		sentence := rest
		if end >= 0 {
			sentence, rest = rest[:end+1], rest[end+1:]
		} else {
			rest = ""
		}
		if s := strings.TrimSpace(sentence); t.Keywords.MatchString(s) {
			return s
		}
	}
	return ""
}
