package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/praetorianer777/kurswechsel/spikes/corpus"
)

func fixture() []corpus.Speech {
	d := func(day int) time.Time { return time.Date(2024, 1, day, 0, 0, 0, 0, time.UTC) }
	return []corpus.Speech{
		{ID: "b", Date: d(2), SpeakerID: "1", Speaker: "A", Party: "X", Paragraphs: []string{"Wir wollen die Wehrpflicht zurück.", "Anderes Thema."}},
		{ID: "a", Date: d(1), SpeakerID: "1", Speaker: "A", Party: "X", Paragraphs: []string{"Die Wehrpflicht bleibt ausgesetzt."}},
		{ID: "c", Date: d(3), SpeakerID: "2", Speaker: "B", Party: "Y", Paragraphs: []string{"Ein freiwilliger Wehrdienst reicht."}},
	}
}

func TestBackends(t *testing.T) {
	for _, b := range Backends(os.Getenv("SPIKE_PG_URL")) {
		t.Run(b.Name(), func(t *testing.T) {
			r, err := Run(context.Background(), b, t.TempDir(), fixture(), "1", []string{"wehrpflicht", "wehrdienst"}, 3)
			if err != nil {
				t.Fatal(err)
			}
			if r.SearchHits != 3 {
				t.Errorf("SearchHits = %d, want 3", r.SearchHits)
			}
			if r.TimelineHits != 2 {
				t.Errorf("TimelineHits = %d, want 2", r.TimelineHits)
			}
			if r.Bytes <= 0 {
				t.Errorf("Bytes = %d, want > 0", r.Bytes)
			}
		})
	}
}

func TestTimelineOrder(t *testing.T) {
	b := &SQLite{}
	ctx := context.Background()
	if err := b.Open(ctx, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err := b.Load(ctx, fixture()); err != nil {
		t.Fatal(err)
	}
	hits, err := b.Timeline(ctx, "1", []string{"wehrpflicht"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].SpeechID != "a" || hits[1].SpeechID != "b" {
		t.Errorf("timeline not in date order: %+v", hits)
	}
}

func TestMedian(t *testing.T) {
	calls := 0
	if _, err := median(5, func() error { calls++; return nil }); err != nil {
		t.Fatal(err)
	}
	if calls != 5 {
		t.Errorf("calls = %d, want 5", calls)
	}
}
