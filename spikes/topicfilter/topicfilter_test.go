package topicfilter

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/spikes/gold"
)

func TestCosine(t *testing.T) {
	if got := Cosine([]float32{1, 0}, []float32{1, 0}); math.Abs(got-1) > 1e-9 {
		t.Errorf("same = %v", got)
	}
	if got := Cosine([]float32{1, 0}, []float32{0, 1}); got != 0 {
		t.Errorf("orthogonal = %v", got)
	}
	if got := Cosine([]float32{0, 0}, []float32{1, 1}); got != 0 {
		t.Errorf("zero = %v", got)
	}
}

// fakeEmbed puts texts mentioning "pflicht" close to the query.
func fakeEmbed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, s := range texts {
		if i == 0 || strings.Contains(strings.ToLower(s), "pflicht") {
			out[i] = []float32{1, 0.1}
		} else {
			out[i] = []float32{0.1, 1}
		}
	}
	return out, nil
}

func TestScoreAndBest(t *testing.T) {
	items := []gold.Item{
		{ID: "1", Text: "Die Wehrpflicht muss zurück.", Relevant: true, Stance: gold.For},
		{ID: "2", Text: "Niemand darf zum Dienst an der Waffe verpflichtet werden.", Relevant: true, Stance: gold.Against},
		{ID: "3", Text: "Die Bundeswehr braucht Panzer.", Relevant: false},
	}
	scored, err := Score(context.Background(), fakeEmbed, items)
	if err != nil {
		t.Fatal(err)
	}
	if !scored[0].Keyword || scored[1].Keyword || scored[2].Keyword {
		t.Errorf("keyword flags wrong: %+v", scored)
	}

	kw := Evaluate(Strategies[0], scored, 0)
	if kw.TP != 1 || kw.FN != 1 || kw.FP != 0 {
		t.Errorf("keywords = %+v", kw)
	}
	_, emb := Best(Strategies[1], scored)
	if emb.F1() != 1 {
		t.Errorf("embedding best F1 = %v, want 1", emb.F1())
	}
}
