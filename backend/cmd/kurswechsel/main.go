// Command kurswechsel ingests Bundestag speeches, classifies stances and
// serves the website.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/api"
	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/demo"
	"github.com/praetorianer777/kurswechsel/internal/ingest"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/web"
)

const usage = `usage: kurswechsel <command> [flags]

commands:
  ingest     download Bundestag protocols and master data into the database
  classify   classify the stance of candidate paragraphs for a topic
  eval       measure a classifier against the hand-labelled gold set
  serve      run the HTTP server
  seed-demo  fill a database with fictional data for development and tests
  sample     draw paragraphs for the training set
  review     check pre-labelled training data in the browser

Run "kurswechsel <command> -h" for the flags of a command.
`

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "kurswechsel:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return errors.New("missing command")
	}
	switch args[0] {
	case "ingest":
		return ingestCmd(ctx, args[1:], stdout)
	case "classify":
		return classifyCmd(ctx, args[1:], stdout)
	case "eval":
		return evalCmd(ctx, args[1:], stdout)
	case "serve":
		return serve(ctx, args[1:])
	case "seed-demo":
		return seedDemo(ctx, args[1:], stdout)
	case "sample":
		return sampleCmd(ctx, args[1:], stdout)
	case "review":
		return reviewCmd(ctx, args[1:], stdout)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		fmt.Fprint(stdout, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address")
	db := fs.String("db", "data/kurswechsel.db", "SQLite database")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, *db)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.NewHandler(&api.Server{Store: st, Frontend: web.Dist()}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", *addr, "db", *db)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func ingestCmd(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	fs.SetOutput(stdout)
	db := fs.String("db", "data/kurswechsel.db", "SQLite database")
	cache := fs.String("cache", "data/raw", "download cache")
	periods := fs.String("periods", "19,20,21", "legislative periods, comma-separated")
	offline := fs.Bool("offline", false, "use only files already in the cache")
	force := fs.Bool("force", false, "re-import sessions already in the database")
	refresh := fs.Bool("refresh-mdb", false, "download the master data again")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ps, err := bundestag.ParsePeriods(*periods)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	st, err := store.Open(ctx, *db)
	if err != nil {
		return err
	}
	defer st.Close()

	f := bundestag.NewFetcher()
	f.Logf = func(format string, a ...any) { slog.Info(fmt.Sprintf(format, a...)) }
	start := time.Now()
	sum, err := ingest.Run(ctx, st, f, ingest.Options{
		CacheDir: *cache, Periods: ps, Offline: *offline, Force: *force, RefreshMdB: *refresh, Logf: f.Logf,
	})
	if err != nil {
		return err
	}
	stats, err := st.Stats(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "imported %d sessions, skipped %d, in %s\n", sum.Imported, sum.Skipped, time.Since(start).Round(time.Second))
	fmt.Fprintf(stdout, "database: %d politicians, %d sessions, %d speeches, %d paragraphs\n",
		stats.Politicians, stats.Sessions, stats.Speeches, stats.Paragraphs)
	return nil
}

func seedDemo(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	fs.SetOutput(stdout)
	db := fs.String("db", "data/demo.db", "SQLite database")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := store.Open(ctx, *db)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := demo.Seed(ctx, st); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "demo data written to %s\n", *db)
	return nil
}
