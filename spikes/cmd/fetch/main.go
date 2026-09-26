// Command fetch downloads the plenary protocols the spikes run on.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/praetorianer777/kurswechsel/spikes/corpus"
)

func main() {
	dir := flag.String("dir", "../data/raw", "download directory")
	flag.Parse()
	client := &http.Client{Timeout: 60 * time.Second}
	for _, p := range []int{19, 20, 21} {
		n, err := corpus.Fetch(context.Background(), client, *dir, p, log.Printf)
		if err != nil {
			log.Fatalf("period %d: %v", p, err)
		}
		log.Printf("period %d: %d protocols", p, n)
	}
}
