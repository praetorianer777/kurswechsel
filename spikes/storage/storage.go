// Package storage compares database engines on the queries Kurswechsel needs:
// bulk loading speeches, keyword search over paragraphs, and a per-speaker
// timeline of matching paragraphs.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/praetorianer777/kurswechsel/spikes/corpus"
)

// Hit is one paragraph on a speaker's timeline.
type Hit struct {
	Date      time.Time
	SpeechID  string
	Paragraph string
}

// Backend is one engine under test.
type Backend interface {
	Name() string
	Open(ctx context.Context, dir string) error
	Load(ctx context.Context, speeches []corpus.Speech) error
	// Search counts paragraphs mentioning any of the terms.
	Search(ctx context.Context, terms []string) (int, error)
	Timeline(ctx context.Context, speakerID string, terms []string) ([]Hit, error)
	// Size is the on-disk footprint in bytes, or -1 when it cannot be measured.
	Size(ctx context.Context) (int64, error)
	Close() error
}

// Result is what one backend measured.
type Result struct {
	Backend      string
	Load         time.Duration
	Search       time.Duration
	SearchHits   int
	Timeline     time.Duration
	TimelineHits int
	Bytes        int64
}

// Run loads the corpus into b and times the queries, each as the median of
// rounds runs so a single cold cache does not decide the outcome.
func Run(ctx context.Context, b Backend, dir string, speeches []corpus.Speech, speakerID string, terms []string, rounds int) (Result, error) {
	r := Result{Backend: b.Name()}
	if err := b.Open(ctx, dir); err != nil {
		return r, fmt.Errorf("%s open: %w", b.Name(), err)
	}
	defer b.Close()

	start := time.Now()
	if err := b.Load(ctx, speeches); err != nil {
		return r, fmt.Errorf("%s load: %w", b.Name(), err)
	}
	r.Load = time.Since(start)

	var err error
	r.Search, err = median(rounds, func() error {
		r.SearchHits, err = b.Search(ctx, terms)
		return err
	})
	if err != nil {
		return r, fmt.Errorf("%s search: %w", b.Name(), err)
	}
	r.Timeline, err = median(rounds, func() error {
		hits, err := b.Timeline(ctx, speakerID, terms)
		r.TimelineHits = len(hits)
		return err
	})
	if err != nil {
		return r, fmt.Errorf("%s timeline: %w", b.Name(), err)
	}
	if r.Bytes, err = b.Size(ctx); err != nil {
		return r, fmt.Errorf("%s size: %w", b.Name(), err)
	}
	return r, nil
}

func median(n int, f func() error) (time.Duration, error) {
	ds := make([]time.Duration, 0, n)
	for range n {
		start := time.Now()
		if err := f(); err != nil {
			return 0, err
		}
		ds = append(ds, time.Since(start))
	}
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0 && ds[j] < ds[j-1]; j-- {
			ds[j], ds[j-1] = ds[j-1], ds[j]
		}
	}
	return ds[len(ds)/2], nil
}

func dirSize(dir string) (int64, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func scanHits(rows *sql.Rows) ([]Hit, error) {
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.Date, &h.SpeechID, &h.Paragraph); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// extraBackends holds engines behind build tags, so the default build and the
// smoke test stay free of cgo.
var extraBackends []func() Backend

// Backends returns every engine compiled into this binary. Postgres joins
// only when pgURL is set, since it needs a running server.
func Backends(pgURL string) []Backend {
	bs := []Backend{&SQLite{}}
	for _, f := range extraBackends {
		bs = append(bs, f())
	}
	if pgURL != "" {
		bs = append(bs, &Postgres{URL: pgURL})
	}
	return bs
}
