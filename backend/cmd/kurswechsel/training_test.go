package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/training"
)

func TestSampleCommand(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "d.db")
	var out bytes.Buffer
	if err := run(context.Background(), []string{"seed-demo", "-db", db}, &out); err != nil {
		t.Fatal(err)
	}
	gold := filepath.Join(dir, "gold.jsonl")
	// Excluded by text although the ID differs, as with the spike parser.
	os.WriteFile(gold, []byte(`{"id":"other#9","text":"Der Wehrdienst muss  freiwillig bleiben, niemand soll gezwungen werden.","relevant":true,"stance":"dagegen"}`+"\n"), 0o644)
	file := filepath.Join(dir, "train.jsonl")
	out.Reset()
	if err := run(context.Background(), []string{"sample", "-db", db, "-exclude", gold, "-out", file, "-n", "100"}, &out); err != nil {
		t.Fatal(err)
	}
	items, err := training.Read(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if strings.HasPrefix(it.Text, "Der Wehrdienst muss freiwillig bleiben, niemand") {
			t.Error("gold paragraph sampled")
		}
	}
	if len(items) == 0 || !strings.Contains(out.String(), "written to") {
		t.Errorf("%d items, output %q", len(items), out.String())
	}
}

func TestReviewCommandMissingFile(t *testing.T) {
	if err := run(context.Background(), []string{"review", "-file", filepath.Join(t.TempDir(), "none.jsonl")}, &bytes.Buffer{}); err == nil {
		t.Error("want error for a missing file")
	}
}
