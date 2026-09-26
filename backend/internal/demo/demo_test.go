package demo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/timeline"
)

func TestSeed(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for range 2 {
		if err := Seed(ctx, st); err != nil {
			t.Fatal(err)
		}
	}
	es, err := st.Timeline(ctx, "90000001", "wehrpflicht")
	if err != nil {
		t.Fatal(err)
	}
	if n := timeline.MarkChanges(es); n != 1 || len(es) != 5 {
		t.Errorf("Musterfrau: %d entries, %d changes; the demo should show one change", len(es), n)
	}
	for _, p := range People {
		if !strings.HasPrefix(p.ID, "9") {
			t.Errorf("demo ID %s could collide with a real MdB ID (11…)", p.ID)
		}
	}
}
