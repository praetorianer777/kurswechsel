package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
			if !strings.Contains(out.String(), "usage: kurswechsel") {
				t.Errorf("usage not printed: %q", out.String())
			}
		})
	}
}
