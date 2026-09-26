package stance

import (
	"fmt"
	"math"
	"strings"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// StyleProbs asks for a single letter and reads the answer's probability
// from the token log-probabilities, so every classification comes with a
// calibrated-looking confidence (issue #25).
const StyleProbs = "probs"

// ProbsPromptVersion identifies the multiple-choice prompt in stored results.
const ProbsPromptVersion = "v3-probs"

// probOptions maps answer letters to outcomes; the order is fixed so the
// prompt and the parser agree.
var probOptions = []struct {
	letter   string
	relevant bool
	stance   string
}{
	{"A", false, Neutral},
	{"B", true, For},
	{"C", true, Against},
	{"D", true, Neutral},
	{"E", true, Unclear},
}

// ProbsSystemPrompt asks for one letter.
func ProbsSystemPrompt(t topic.Topic) string {
	return `You classify paragraphs from speeches in the German Bundestag.

` + t.Definition + `

Answer with exactly one letter:
A = not relevant
B = dafuer
C = dagegen
D = neutral
E = unklar

Decide from the paragraph alone and judge only the speaker's own position; do not use your knowledge of the speaker or their party. Reply with the letter only.`
}

type logprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

// letterProbs turns the first answer token's alternatives into a probability
// per letter, normalised over the five letters. Variants such as " b" count
// for "B".
func letterProbs(top []logprob) (map[string]float64, error) {
	probs := map[string]float64{}
	var total float64
	for _, lp := range top {
		l := strings.ToUpper(strings.TrimSpace(lp.Token))
		for _, o := range probOptions {
			if l == o.letter {
				p := math.Exp(lp.Logprob)
				probs[l] += p
				total += p
			}
		}
	}
	if total == 0 {
		return nil, fmt.Errorf("no answer letter among the likely tokens")
	}
	for l := range probs {
		probs[l] /= total
	}
	return probs, nil
}

// probsAnswer picks the most likely letter. A side below minConfidence
// becomes "unklar": with an uncertain model, no side is better than a wrong
// one, since a wrong side draws a false change of position.
func probsAnswer(probs map[string]float64, minConfidence float64, paragraph string, t topic.Topic) Answer {
	best := probOptions[0]
	for _, o := range probOptions[1:] {
		if probs[o.letter] > probs[best.letter] {
			best = o
		}
	}
	p := probs[best.letter]
	a := Answer{Relevant: best.relevant, Stance: best.stance, Confidence: p}
	if !a.Relevant {
		return a
	}
	if minConfidence > 0 && (a.Stance == For || a.Stance == Against) && p < minConfidence {
		a.Stance = Unclear
	}
	a.Quote = QuoteFor(paragraph, t)
	a.Rationale = fmt.Sprintf("Automatisch eingeordnet mit einer Wahrscheinlichkeit von %.0f %%.", 100*p)
	return a
}
