// Package ingest loads the Bundestag's open data into the store.
package ingest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/store"
)

// Options controls one ingest run.
type Options struct {
	CacheDir string
	Periods  []int
	// Offline uses only files already in CacheDir.
	Offline bool
	// Force re-imports sessions that are already in the database.
	Force      bool
	RefreshMdB bool
	Logf       func(format string, args ...any)
	Now        func() time.Time
}

// Summary reports what a run did.
type Summary struct {
	Politicians int
	Imported    int
	Skipped     int
}

// Run imports the master data first, so speeches can be matched to factions,
// then every session of the requested periods.
func Run(ctx context.Context, st *store.Store, f *bundestag.Fetcher, o Options) (Summary, error) {
	var sum Summary
	logf := o.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	minPeriod := o.Periods[0]
	for _, p := range o.Periods {
		minPeriod = min(minPeriod, p)
	}

	mdbPath := bundestag.MdBPath(o.CacheDir)
	if !o.Offline {
		var err error
		if mdbPath, err = f.FetchMdB(ctx, o.CacheDir, o.RefreshMdB); err != nil {
			return sum, fmt.Errorf("master data: %w", err)
		}
	}
	politicians, err := bundestag.ReadMdBZip(mdbPath, minPeriod)
	if err != nil {
		return sum, fmt.Errorf("master data: %w", err)
	}
	if err := st.UpsertPoliticians(ctx, politicians); err != nil {
		return sum, err
	}
	sum.Politicians = len(politicians)
	logf("master data: %d members since period %d", len(politicians), minPeriod)

	for _, period := range o.Periods {
		paths, err := protocolPaths(ctx, f, o, period)
		if err != nil {
			return sum, fmt.Errorf("period %d: %w", period, err)
		}
		for i, path := range paths {
			session := i + 1
			if !o.Force {
				done, err := st.HasSession(ctx, period, session)
				if err != nil {
					return sum, err
				}
				if done {
					sum.Skipped++
					continue
				}
			}
			if err := importFile(ctx, st, path, f.ProtocolURL(period, session), now()); err != nil {
				return sum, fmt.Errorf("%s: %w", path, err)
			}
			sum.Imported++
		}
		logf("period %d: %d sessions", period, len(paths))
	}
	return sum, nil
}

func protocolPaths(ctx context.Context, f *bundestag.Fetcher, o Options, period int) ([]string, error) {
	if !o.Offline {
		return f.FetchProtocols(ctx, o.CacheDir, period)
	}
	var paths []string
	for session := 1; ; session++ {
		p := bundestag.ProtocolPath(o.CacheDir, period, session)
		if _, err := os.Stat(p); err != nil {
			return paths, nil
		}
		paths = append(paths, p)
	}
}

func importFile(ctx context.Context, st *store.Store, path, url string, now time.Time) error {
	fh, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	p, err := bundestag.ParseProtocol(fh)
	if err != nil {
		return err
	}
	return st.SaveProtocol(ctx, p, url, now)
}
