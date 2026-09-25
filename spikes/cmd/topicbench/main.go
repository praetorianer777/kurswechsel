// Command topicbench scores the topic filter strategies on the gold set and
// prints a Markdown table for the ADR.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/praetorianer777/kurswechsel/spikes/gold"
	"github.com/praetorianer777/kurswechsel/spikes/ollama"
	"github.com/praetorianer777/kurswechsel/spikes/topicfilter"
)

func main() {
	goldPath := flag.String("gold", "testdata/gold-wehrpflicht.jsonl", "gold set")
	host := flag.String("ollama", "http://127.0.0.1:11434", "Ollama URL")
	model := flag.String("model", "bge-m3", "embedding model")
	flag.Parse()

	f, err := os.Open(*goldPath)
	if err != nil {
		log.Fatal(err)
	}
	items, err := gold.Read(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	client := ollama.New(*host)
	embed := func(ctx context.Context, texts []string) ([][]float32, error) {
		return client.Embed(ctx, *model, texts)
	}
	start := time.Now()
	scored, err := topicfilter.Score(context.Background(), embed, items)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("embedded %d texts with %s in %s", len(items)+1, *model, time.Since(start).Round(time.Millisecond))

	fmt.Println("| Strategy | Threshold | Precision | Recall | F1 | FP | FN |")
	fmt.Println("| --- | ---: | ---: | ---: | ---: | ---: | ---: |")
	for _, st := range topicfilter.Strategies {
		t, b := topicfilter.Best(st, scored)
		th := "–"
		if st.Tuned {
			th = fmt.Sprintf("%.3f", t)
		}
		fmt.Printf("| %s | %s | %.2f | %.2f | %.2f | %d | %d |\n", st.Name, th, b.Precision(), b.Recall(), b.F1(), b.FP, b.FN)
	}
}
