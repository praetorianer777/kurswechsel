package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/praetorianer777/kurswechsel/internal/classify"
	"github.com/praetorianer777/kurswechsel/internal/eval"
	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

// DefaultModel is the local model ADR 0004 selected.
const DefaultModel = "qwen3:30b-a3b"

type classifierFlags struct {
	provider, model, ollama, topic string
	think                          bool
}

func (c *classifierFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&c.provider, "provider", "ollama", "ollama, claude (paid API, opt-in) or fake")
	fs.StringVar(&c.model, "model", DefaultModel, "model name; for -provider claude e.g. claude-opus-5")
	fs.BoolVar(&c.think, "think", false, "leave the Ollama model's thinking on (needed for qwen3.5 and gpt-oss)")
	fs.StringVar(&c.ollama, "ollama", "http://127.0.0.1:11434", "Ollama URL")
	fs.StringVar(&c.topic, "topic", topic.Wehrpflicht.Slug, "topic slug")
}

func (c *classifierFlags) build() (stance.Classifier, topic.Topic, error) {
	t, ok := topic.Get(c.topic)
	if !ok {
		return nil, t, fmt.Errorf("unknown topic %q", c.topic)
	}
	switch c.provider {
	case "ollama":
		return &stance.Ollama{BaseURL: c.ollama, Model: c.model, Think: c.think}, t, nil
	case "claude":
		if c.model == DefaultModel {
			return nil, t, fmt.Errorf("-provider claude needs -model, e.g. claude-opus-5")
		}
		return &stance.Claude{Client: anthropic.NewClient(), Model: c.model}, t, nil
	case "fake":
		return stance.Fake{}, t, nil
	}
	return nil, t, fmt.Errorf("unknown provider %q", c.provider)
}

func classifyCmd(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	fs.SetOutput(stdout)
	db := fs.String("db", "data/kurswechsel.db", "SQLite database")
	limit := fs.Int("limit", 0, "classify at most this many paragraphs (0 = all)")
	redo := fs.Bool("redo", false, "discard earlier classifications of the topic first")
	var cf classifierFlags
	cf.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, t, err := cf.build()
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

	start := time.Now()
	sum, err := classify.Run(ctx, st, c, t, classify.Options{
		Limit: *limit, Redo: *redo,
		Logf: func(format string, a ...any) { slog.Info(fmt.Sprintf(format, a...)) },
	})
	fmt.Fprintf(stdout, "%s: %d candidates, %d classified (%d relevant), %d failed, in %s\n",
		t.Slug, sum.Candidates, sum.Classified, sum.Relevant, sum.Failed, time.Since(start).Round(time.Second))
	return err
}

func evalCmd(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(stdout)
	goldPath := fs.String("gold", "../evaluation/gold-wehrpflicht.jsonl", "gold set (JSON Lines)")
	jsonOut := fs.String("json", "", "also write the report as JSON to this file")
	var cf classifierFlags
	cf.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, t, err := cf.build()
	if err != nil {
		return err
	}
	f, err := os.Open(*goldPath)
	if err != nil {
		return err
	}
	items, err := eval.ReadGold(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", *goldPath, err)
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	r, err := eval.Run(ctx, c, t, items, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := r.WriteMarkdown(stdout); err != nil {
		return err
	}
	if *jsonOut == "" {
		return nil
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*jsonOut, append(b, '\n'), 0o644)
}
