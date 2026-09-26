// Command goldcandidates draws a reproducible sample of paragraphs for hand
// labelling from three strata: explicit conscription vocabulary, talk of
// duties and service without that vocabulary (where the narrow keywords would
// miss relevant paragraphs), and the wider defence debate (hard negatives).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"regexp"

	"github.com/praetorianer777/kurswechsel/spikes/corpus"
	"github.com/praetorianer777/kurswechsel/spikes/gold"
)

var (
	narrow = regexp.MustCompile(`(?i)wehrpflicht|wehrdienst|musterung|dienstpflicht|pflichtdienst|gesellschaftsjahr|pflichtjahr`)
	wide   = regexp.MustCompile(`(?i)bundeswehr|soldat|reservist|personal(gewinnung|lage|mangel)|verteidigungsfähig|kriegstüchtig|freiwillig`)
	duty   = regexp.MustCompile(`(?i)(verpflicht|zwang|pflicht)\w*.{0,80}(dienst|jahrgang|junge männer|junge menschen|jugendliche)|(dienst|jahrgang|junge männer|junge menschen|jugendliche).{0,80}(verpflicht|zwang|pflicht)`)
)

func main() {
	raw := flag.String("raw", "../data/raw", "downloaded protocols")
	n := flag.Int("n", 45, "items in the narrow and wide strata")
	nDuty := flag.Int("duty", 20, "items in the duty stratum")
	minLen := flag.Int("min", 120, "minimum paragraph length in characters")
	flag.Parse()

	speeches, err := corpus.LoadDir(*raw)
	if err != nil {
		log.Fatal(err)
	}
	var hits, duties, near []gold.Item
	for _, s := range speeches {
		for i, p := range s.Paragraphs {
			if len(p) < *minLen {
				continue
			}
			it := gold.Item{
				ID: fmt.Sprintf("%s#%d", s.ID, i), SpeechID: s.ID, Date: s.Date.Format("2006-01-02"),
				Speaker: s.Speaker, Party: s.Party, Text: p,
			}
			if narrow.MatchString(p) {
				hits = append(hits, it)
				continue
			}
			if wide.MatchString(p) {
				near = append(near, it)
			}
			if duty.MatchString(p) {
				duties = append(duties, it)
			}
		}
	}
	log.Printf("narrow: %d, duty: %d, wide: %d", len(hits), len(duties), len(near))

	r := rand.New(rand.NewPCG(2026, 9))
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	strata := []struct {
		pool []gold.Item
		n    int
	}{{hits, *n}, {near, *n}, {duties, *nDuty}}
	for _, st := range strata {
		pool := st.pool
		r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		for _, it := range pool[:min(st.n, len(pool))] {
			if err := enc.Encode(it); err != nil {
				log.Fatal(err)
			}
		}
	}
}
