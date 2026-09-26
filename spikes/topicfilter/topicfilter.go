// Package topicfilter compares ways of deciding whether a paragraph is about
// a topic: a keyword regex, embedding similarity, and combinations of both.
package topicfilter

import (
	"context"
	"math"
	"regexp"
	"sort"

	"github.com/praetorianer777/kurswechsel/spikes/gold"
)

// Keywords is the conscription prefilter under test.
var Keywords = regexp.MustCompile(`(?i)wehrpflicht|wehrdienst|musterung|dienstpflicht|pflichtdienst|gesellschaftsjahr|pflichtjahr`)

// Query describes the topic for the embedding model.
const Query = "Soll es in Deutschland eine Wehrpflicht, einen verpflichtenden Wehrdienst, eine Musterungspflicht oder eine allgemeine Dienstpflicht geben?"

// Embedder turns texts into vectors.
type Embedder func(ctx context.Context, texts []string) ([][]float32, error)

// Scored is one gold item with its similarity to the topic query.
type Scored struct {
	Item    gold.Item
	Keyword bool
	Sim     float64
}

// Score embeds the query and every item.
func Score(ctx context.Context, embed Embedder, items []gold.Item) ([]Scored, error) {
	texts := make([]string, 0, len(items)+1)
	texts = append(texts, Query)
	for _, it := range items {
		texts = append(texts, it.Text)
	}
	vecs, err := embed(ctx, texts)
	if err != nil {
		return nil, err
	}
	out := make([]Scored, len(items))
	for i, it := range items {
		out[i] = Scored{Item: it, Keyword: Keywords.MatchString(it.Text), Sim: Cosine(vecs[0], vecs[i+1])}
	}
	return out, nil
}

// Cosine similarity; zero vectors score 0.
func Cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// Strategy decides relevance from a scored item and a similarity threshold.
type Strategy struct {
	Name   string
	Decide func(s Scored, threshold float64) bool
	// Tuned strategies depend on the threshold; the keyword baseline does not.
	Tuned bool
}

// Strategies lists the approaches the ADR compares.
var Strategies = []Strategy{
	{Name: "keywords", Decide: func(s Scored, _ float64) bool { return s.Keyword }},
	{Name: "embedding", Tuned: true, Decide: func(s Scored, t float64) bool { return s.Sim >= t }},
	{Name: "keywords AND embedding", Tuned: true, Decide: func(s Scored, t float64) bool { return s.Keyword && s.Sim >= t }},
	{Name: "keywords OR embedding", Tuned: true, Decide: func(s Scored, t float64) bool { return s.Keyword || s.Sim >= t }},
}

// Evaluate scores a strategy at one threshold.
func Evaluate(st Strategy, scored []Scored, threshold float64) gold.Binary {
	var b gold.Binary
	for _, s := range scored {
		b.Add(s.Item.Relevant, st.Decide(s, threshold))
	}
	return b
}

// Best finds the threshold with the highest F1, trying every observed
// similarity. It is tuned on the gold set itself, so it is an upper bound.
func Best(st Strategy, scored []Scored) (float64, gold.Binary) {
	if !st.Tuned {
		return 0, Evaluate(st, scored, 0)
	}
	cands := make([]float64, len(scored))
	for i, s := range scored {
		cands[i] = s.Sim
	}
	sort.Float64s(cands)
	bestT, best := 0.0, gold.Binary{}
	bestF1 := -1.0
	for _, t := range cands {
		b := Evaluate(st, scored, t)
		if f := b.F1(); f > bestF1 {
			bestT, best, bestF1 = t, b, f
		}
	}
	return bestT, best
}
