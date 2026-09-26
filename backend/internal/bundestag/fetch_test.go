package bundestag

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func testFetcher(url string) *Fetcher {
	f := NewFetcher()
	f.HTTP = http.DefaultClient
	f.ProtocolBase = url + "/btp"
	f.MdBURL = url + "/mdb.zip"
	f.Delay = 0
	return f
}

func TestFetchProtocols(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.Header.Get("User-Agent"); got != UserAgent {
			t.Errorf("User-Agent = %q", got)
		}
		switch r.URL.Path {
		case "/btp/20/20001.xml", "/btp/20/20002.xml", "/btp/20/20003.xml":
			w.Write([]byte("<xml/>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	// Session 1 is already cached and must not be downloaded again.
	cached := ProtocolPath(dir, 20, 1)
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cached, []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := testFetcher(srv.URL).FetchProtocols(context.Background(), dir, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("paths = %v", paths)
	}
	if hits.Load() != 3 {
		t.Errorf("requests = %d, want 3 (sessions 2, 3 and the 404 for 4)", hits.Load())
	}
	if b, _ := os.ReadFile(cached); string(b) != "cached" {
		t.Error("cached file was overwritten")
	}
	if b, _ := os.ReadFile(paths[2]); string(b) != "<xml/>" {
		t.Errorf("session 3 = %q", b)
	}
}

func TestFetchServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	dir := t.TempDir()
	if _, err := testFetcher(srv.URL).FetchProtocols(context.Background(), dir, 20); err == nil {
		t.Fatal("want error")
	}
	if entries, _ := os.ReadDir(filepath.Join(dir, "20")); len(entries) != 0 {
		t.Errorf("left files behind: %v", entries)
	}
}

func TestFetchMdB(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Write([]byte("zip"))
	}))
	defer srv.Close()

	f, dir := testFetcher(srv.URL), t.TempDir()
	for _, refresh := range []bool{false, false, true} {
		if _, err := f.FetchMdB(context.Background(), dir, refresh); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 2 {
		t.Errorf("downloads = %d, want 2 (first call and refresh)", hits.Load())
	}
}

func TestFetchCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("x")) }))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := testFetcher(srv.URL).FetchProtocols(ctx, t.TempDir(), 20); err == nil {
		t.Fatal("want error from cancelled context")
	}
}
