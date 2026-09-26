package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/eval"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/topic"
	"github.com/praetorianer777/kurswechsel/internal/training"
)

func sampleCmd(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("sample", flag.ContinueOnError)
	fs.SetOutput(stdout)
	db := fs.String("db", "data/kurswechsel.db", "SQLite database with classified candidates")
	slug := fs.String("topic", topic.Wehrpflicht.Slug, "topic slug")
	n := fs.Int("n", 400, "number of paragraphs")
	goldPath := fs.String("exclude", "../evaluation/gold-wehrpflicht.jsonl", "gold set whose paragraphs must not be sampled")
	out := fs.String("out", "../training/wehrpflicht.jsonl", "output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	f, err := os.Open(*goldPath)
	if err != nil {
		return err
	}
	gold, err := eval.ReadGold(f)
	f.Close()
	if err != nil {
		return err
	}
	// The gold set was drawn with the spike parser, whose paragraph numbers
	// can differ from production, so paragraphs are also excluded by text.
	exclude := map[string]bool{}
	goldText := map[string]bool{}
	for _, g := range gold {
		exclude[g.ID] = true
		goldText[strings.Join(strings.Fields(g.Text), " ")] = true
	}
	st, err := store.Open(ctx, *db)
	if err != nil {
		return err
	}
	defer st.Close()
	cands, err := st.Candidates(ctx, *slug)
	if err != nil {
		return err
	}
	for _, c := range cands {
		if goldText[strings.Join(strings.Fields(c.Text), " ")] {
			exclude[c.ID] = true
		}
	}
	picked := training.Sample(cands, *n, exclude, 2027)
	items := make([]training.Item, len(picked))
	for i, c := range picked {
		items[i] = training.Item{ID: c.ID, SpeechID: c.SpeechID, Date: c.Date, Speaker: c.Speaker, Party: c.Faction, Text: c.Text}
	}
	if err := training.Write(*out, items); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%d of %d candidates written to %s (%d gold paragraphs excluded)\n", len(items), len(cands), *out, len(gold))
	return nil
}

func reviewCmd(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(stdout)
	path := fs.String("file", "../training/wehrpflicht.jsonl", "training file with pre-labels")
	addr := fs.String("addr", "127.0.0.1:8090", "listen address")
	sample := fs.Int("sample", 50, "sure pre-labels to check for the agreement rate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rv, err := training.NewReviewer(*path, *sample)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv := &http.Server{Addr: *addr, Handler: rv.Handler(), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("review page", "url", "http://"+*addr, "file", *path)
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	}
}
