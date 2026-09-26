// Command stancebench runs local Ollama models over the paragraphs the keyword
// prefilter passes and scores relevance, stance and quote fidelity against
// the gold set. It never calls a paid API.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/praetorianer777/kurswechsel/spikes/gold"
	"github.com/praetorianer777/kurswechsel/spikes/ollama"
	"github.com/praetorianer777/kurswechsel/spikes/stance"
	"github.com/praetorianer777/kurswechsel/spikes/topicfilter"
)

type row struct {
	Model    string
	Relevant gold.Binary
	Stance   gold.Confusion
	Verbatim int
	N        int
	Errors   int
	Nanos    []int64
	Tokens   int
	Wall     time.Duration
}

func main() {
	goldPath := flag.String("gold", "../evaluation/gold-wehrpflicht.jsonl", "gold set")
	host := flag.String("ollama", "http://127.0.0.1:11434", "Ollama URL")
	models := flag.String("models", "qwen3.5:9b@think", "comma-separated Ollama models; append @think to leave thinking on")
	dump := flag.String("dump", "", "write every answer as JSON Lines to this file")
	flag.Parse()

	f, err := os.Open(*goldPath)
	if err != nil {
		log.Fatal(err)
	}
	all, err := gold.Read(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	var items []gold.Item
	for _, it := range all {
		if topicfilter.Keywords.MatchString(it.Text) {
			items = append(items, it)
		}
	}
	log.Printf("%d of %d gold items pass the keyword prefilter", len(items), len(all))

	var out *json.Encoder
	if *dump != "" {
		df, err := os.Create(*dump)
		if err != nil {
			log.Fatal(err)
		}
		defer df.Close()
		out = json.NewEncoder(df)
		out.SetEscapeHTML(false)
	}

	client := ollama.New(*host)
	var rows []row
	for _, spec := range strings.Split(*models, ",") {
		m := stance.ParseModel(spec)
		r := row{Model: m.String(), Stance: gold.Confusion{}}
		start := time.Now()
		for _, it := range items {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			res, err := stance.Classify(ctx, client, m, it)
			cancel()
			if err != nil {
				log.Printf("%s %s: %v", m, it.ID, err)
				r.Errors++
				continue
			}
			r.N++
			r.Nanos = append(r.Nanos, res.Nanos)
			r.Tokens += res.OutputTokens
			r.Relevant.Add(it.Relevant, res.Relevant)
			if it.Relevant {
				got := res.Stance
				if !res.Relevant {
					got = gold.Neutral
				}
				r.Stance.Add(it.Stance, got)
			}
			if res.QuoteVerbatim {
				r.Verbatim++
			}
			if out != nil {
				_ = out.Encode(map[string]any{"model": m.String(), "id": it.ID, "gold_relevant": it.Relevant, "gold_stance": it.Stance, "answer": res})
			}
		}
		r.Wall = time.Since(start)
		log.Printf("%s done in %s", m, r.Wall.Round(time.Second))
		rows = append(rows, r)
	}

	fmt.Println("| Model | Relevance F1 | Stance accuracy | Stance macro-F1 | „unklar“ rate | Verbatim quotes | Median latency | Errors |")
	fmt.Println("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
	for _, r := range rows {
		fmt.Printf("| %s | %.2f | %.2f | %.2f | %.0f %% | %.0f %% | %.1f s | %d |\n", r.Model,
			r.Relevant.F1(), r.Stance.Accuracy(), r.Stance.MacroF1(), 100*r.Stance.Rate(gold.Unclear),
			100*float64(r.Verbatim)/float64(max(r.N, 1)), medianSeconds(r.Nanos), r.Errors)
	}
}

func medianSeconds(ns []int64) float64 {
	if len(ns) == 0 {
		return 0
	}
	s := append([]int64(nil), ns...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return float64(s[len(s)/2]) / 1e9
}
