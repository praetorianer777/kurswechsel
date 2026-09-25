package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifierFlags(t *testing.T) {
	for _, tc := range []struct {
		flags   classifierFlags
		name    string
		wantErr string
	}{
		{flags: classifierFlags{provider: "ollama", model: "m", topic: "wehrpflicht"}, name: "ollama/m"},
		{flags: classifierFlags{provider: "fake", topic: "wehrpflicht"}, name: "fake"},
		{flags: classifierFlags{provider: "claude", model: "claude-opus-5", topic: "wehrpflicht"}, name: "claude/claude-opus-5"},
		{flags: classifierFlags{provider: "claude", model: DefaultModel, topic: "wehrpflicht"}, wantErr: "needs -model"},
		{flags: classifierFlags{provider: "openai", topic: "wehrpflicht"}, wantErr: `unknown provider "openai"`},
		{flags: classifierFlags{provider: "fake", topic: "mietpreise"}, wantErr: `unknown topic "mietpreise"`},
	} {
		c, _, err := tc.flags.build()
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("%+v: err = %v, want %q", tc.flags, err, tc.wantErr)
			}
			continue
		}
		if err != nil || c.Name() != tc.name {
			t.Errorf("%+v: got %v, %v", tc.flags, c, err)
		}
	}
}

func TestEvalCommand(t *testing.T) {
	dir := t.TempDir()
	gold := filepath.Join(dir, "gold.jsonl")
	os.WriteFile(gold, []byte(`{"id":"1","text":"Wir müssen die Wehrpflicht wieder einführen.","relevant":true,"stance":"dafuer"}
{"id":"2","text":"Die Bundeswehr braucht Panzer.","relevant":false}
`), 0o644)
	out := filepath.Join(dir, "report.json")
	var buf bytes.Buffer
	if err := run(context.Background(), []string{"eval", "-provider", "fake", "-gold", gold, "-json", out}, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "| Stance accuracy | 1.00 |") {
		t.Errorf("output:\n%s", buf.String())
	}
	var r map[string]any
	b, _ := os.ReadFile(out)
	if err := json.Unmarshal(b, &r); err != nil || r["classifier"] != "fake" {
		t.Errorf("json report = %s, %v", b, err)
	}
	if err := run(context.Background(), []string{"eval", "-provider", "fake", "-gold", filepath.Join(dir, "missing")}, &buf); err == nil {
		t.Error("want error for missing gold set")
	}
}

func TestClassifyCommand(t *testing.T) {
	var buf bytes.Buffer
	db := filepath.Join(t.TempDir(), "k.db")
	if err := run(context.Background(), []string{"classify", "-provider", "fake", "-db", db}, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "wehrpflicht: 0 candidates") {
		t.Errorf("output: %s", buf.String())
	}
}
