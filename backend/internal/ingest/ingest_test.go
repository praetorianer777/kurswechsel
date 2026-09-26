package ingest

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/store"
)

func mdbZip(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../bundestag/testdata/mdb.xml")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("MDB_STAMMDATEN.XML")
	w.Write(data)
	zw.Close()
	return buf.Bytes()
}

func server(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	proto, err := os.ReadFile("../bundestag/testdata/edge-cases.xml")
	if err != nil {
		t.Fatal(err)
	}
	proto = bytes.Replace(proto, []byte(`sitzung-nr="42"`), []byte(`sitzung-nr="1"`), 1)
	zipped := mdbZip(t)
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/mdb.zip":
			w.Write(zipped)
		case "/btp/21/21001.xml":
			w.Write(proto)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

func fetcher(url string) *bundestag.Fetcher {
	f := bundestag.NewFetcher()
	f.HTTP = http.DefaultClient
	f.ProtocolBase = url + "/btp"
	f.MdBURL = url + "/mdb.zip"
	f.Delay = 0
	return f
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	srv, requests := server(t)
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cache := t.TempDir()
	var log []string
	opts := Options{
		CacheDir: cache, Periods: []int{20, 21},
		Logf: func(f string, a ...any) { log = append(log, f) },
		Now:  func() time.Time { return time.Unix(0, 0) },
	}

	sum, err := Run(ctx, st, fetcher(srv.URL), opts)
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{Politicians: 1, Imported: 1}) {
		t.Errorf("first run = %+v", sum)
	}
	stats, _ := st.Stats(ctx)
	if stats.Speeches != 2 || stats.Paragraphs != 6 {
		t.Errorf("stats = %+v", stats)
	}
	if len(log) == 0 {
		t.Error("nothing logged")
	}

	first := *requests
	sum, err = Run(ctx, st, fetcher(srv.URL), opts)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Imported != 0 || sum.Skipped != 1 {
		t.Errorf("second run = %+v, want the session skipped", sum)
	}
	// Cached files, the master data included, are not downloaded again; each
	// period only probes for its next session.
	if got := *requests - first; got != 2 {
		t.Errorf("second run made %d requests, want 2", got)
	}

	opts.Offline, opts.Force = true, true
	sum, err = Run(ctx, st, fetcher("http://127.0.0.1:1"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Imported != 1 {
		t.Errorf("offline forced run = %+v", sum)
	}
}

func TestRunMissingMasterData(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_, err = Run(ctx, st, fetcher("http://127.0.0.1:1"), Options{CacheDir: t.TempDir(), Periods: []int{21}, Offline: true})
	if err == nil || !strings.Contains(err.Error(), "master data") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunBrokenProtocol(t *testing.T) {
	ctx := context.Background()
	srv, _ := server(t)
	cache := t.TempDir()
	broken := bundestag.ProtocolPath(cache, 21, 1)
	os.MkdirAll(filepath.Dir(broken), 0o755)
	os.WriteFile(broken, []byte("<nope"), 0o644)

	st, _ := store.Open(ctx, ":memory:")
	defer st.Close()
	_, err := Run(ctx, st, fetcher(srv.URL), Options{CacheDir: cache, Periods: []int{21}})
	if err == nil || !strings.Contains(err.Error(), "21001.xml") {
		t.Fatalf("err = %v", err)
	}
}
