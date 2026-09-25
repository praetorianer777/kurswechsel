// Command storagebench loads the downloaded protocols into every compiled-in
// engine and prints a Markdown table for the storage ADR.
//
//	go run -tags duckdb ./cmd/storagebench -pg postgres://...
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/praetorianer777/kurswechsel/spikes/corpus"
	"github.com/praetorianer777/kurswechsel/spikes/storage"
)

func main() {
	raw := flag.String("raw", "../data/raw", "downloaded protocols")
	pg := flag.String("pg", "", "PostgreSQL URL (optional)")
	terms := flag.String("terms", "wehrpflicht,wehrdienst", "comma-separated search terms")
	rounds := flag.Int("rounds", 7, "query repetitions (median)")
	flag.Parse()

	start := time.Now()
	speeches, err := corpus.LoadDir(*raw)
	if err != nil {
		log.Fatal(err)
	}
	paragraphs := 0
	for _, s := range speeches {
		paragraphs += len(s.Paragraphs)
	}
	log.Printf("parsed %d speeches, %d paragraphs in %s", len(speeches), paragraphs, time.Since(start).Round(time.Millisecond))

	ts := strings.Split(*terms, ",")
	speaker := topSpeaker(speeches, ts)
	log.Printf("timeline speaker: %s", speaker)

	fmt.Println("| Engine | Load | Search | Hits | Timeline | Hits | Size |")
	fmt.Println("| --- | ---: | ---: | ---: | ---: | ---: | ---: |")
	for _, b := range storage.Backends(*pg) {
		dir, err := os.MkdirTemp("", "storagebench")
		if err != nil {
			log.Fatal(err)
		}
		r, err := storage.Run(context.Background(), b, dir, speeches, speaker, ts, *rounds)
		os.RemoveAll(dir)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("| %s | %s | %s | %d | %s | %d | %.0f MB |\n", r.Backend,
			r.Load.Round(time.Millisecond), r.Search.Round(10*time.Microsecond), r.SearchHits,
			r.Timeline.Round(10*time.Microsecond), r.TimelineHits, float64(r.Bytes)/1e6)
	}
}

// topSpeaker picks the person who mentions the terms most, so the timeline
// query returns a realistic number of rows.
func topSpeaker(speeches []corpus.Speech, terms []string) string {
	counts := map[string]int{}
	for _, s := range speeches {
		for _, p := range s.Paragraphs {
			lp := strings.ToLower(p)
			for _, t := range terms {
				if strings.Contains(lp, t) {
					counts[s.SpeakerID+" "+s.Speaker]++
					break
				}
			}
		}
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return counts[keys[i]] > counts[keys[j]] })
	if len(keys) == 0 {
		return ""
	}
	id, _, _ := strings.Cut(keys[0], " ")
	return id
}
