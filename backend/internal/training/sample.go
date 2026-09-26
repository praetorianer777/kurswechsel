// Package training builds the labelled training set: a reproducible sample
// of candidate paragraphs, pre-labels, and a local page for reviewing them.
package training

import (
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/praetorianer777/kurswechsel/internal/store"
)

// Sample draws n candidates, skipping excluded IDs (the gold set), spread
// evenly over legislative period and faction so that no single debate or
// party dominates. The same seed always gives the same sample.
func Sample(cands []store.CandidateInfo, n int, exclude map[string]bool, seed uint64) []store.CandidateInfo {
	groups := map[string][]store.CandidateInfo{}
	var keys []string
	for _, c := range cands {
		if exclude[c.ID] {
			continue
		}
		k := fmt.Sprintf("%d|%s", c.Period, c.Faction)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], c)
	}
	sort.Strings(keys)
	r := rand.New(rand.NewPCG(seed, 27))
	for _, k := range keys {
		g := groups[k]
		r.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
	}
	var out []store.CandidateInfo
	for len(out) < n {
		added := false
		for _, k := range keys {
			if len(groups[k]) == 0 || len(out) == n {
				continue
			}
			out = append(out, groups[k][0])
			groups[k] = groups[k][1:]
			added = true
		}
		if !added {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
