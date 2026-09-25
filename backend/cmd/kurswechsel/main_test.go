package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "no command", args: nil, wantErr: "missing command"},
		{name: "unknown command", args: []string{"nope"}, wantErr: `unknown command "nope"`},
		{name: "help", args: []string{"help"}},
		{name: "ingest rejects old periods", args: []string{"ingest", "-periods", "18"}, wantErr: `legislative period "18": only 19 and later have structured protocols`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(context.Background(), tt.args, &out)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
			if tt.args != nil && tt.args[0] == "ingest" {
				return
			}
			if !strings.Contains(out.String(), "usage: kurswechsel") {
				t.Errorf("usage not printed: %q", out.String())
			}
		})
	}
}

func TestIngestOffline(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := run(context.Background(), []string{"ingest", "-db", dir + "/k.db", "-cache", dir, "-periods", "21", "-offline"}, &out)
	if err == nil || !strings.Contains(err.Error(), "master data") {
		t.Fatalf("err = %v, want missing master data", err)
	}
}

func TestServeBadFlag(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"serve", "-nope"}, &out); err == nil {
		t.Fatal("want flag error")
	}
}

func TestServeShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{"serve", "-addr", "127.0.0.1:0", "-db", t.TempDir() + "/k.db"}, &bytes.Buffer{})
	}()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop")
	}
}
