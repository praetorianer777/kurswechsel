package bundestag

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Default locations of the open data.
const (
	DefaultProtocolBase = "https://dserver.bundestag.de/btp"
	DefaultMdBURL       = "https://www.bundestag.de/resource/blob/472878/MdB-Stammdaten.zip"
	UserAgent           = "kurswechsel/0.1 (+https://github.com/praetorianer777/kurswechsel)"
)

// ErrNotFound means the server has no such file.
var ErrNotFound = errors.New("not found")

// Fetcher downloads open data files into a local cache, once each.
type Fetcher struct {
	HTTP         *http.Client
	ProtocolBase string
	MdBURL       string
	// Delay between downloads, to stay polite towards the Bundestag's servers.
	Delay time.Duration
	Logf  func(format string, args ...any)
}

// NewFetcher returns a Fetcher for the live Bundestag servers.
func NewFetcher() *Fetcher {
	return &Fetcher{
		HTTP:         &http.Client{Timeout: 2 * time.Minute},
		ProtocolBase: DefaultProtocolBase,
		MdBURL:       DefaultMdBURL,
		Delay:        300 * time.Millisecond,
		Logf:         func(string, ...any) {},
	}
}

// ProtocolURL is where the XML of one session is published.
func (f *Fetcher) ProtocolURL(period, session int) string {
	return fmt.Sprintf("%s/%d/%d%03d.xml", f.ProtocolBase, period, period, session)
}

// ProtocolPath is where a session's XML is cached below dir.
func ProtocolPath(dir string, period, session int) string {
	return filepath.Join(dir, fmt.Sprint(period), fmt.Sprintf("%d%03d.xml", period, session))
}

// FetchProtocols downloads every session of a period that is not cached yet
// and returns the paths of all sessions, in order. Sessions are numbered
// without gaps, so the first missing one ends the period.
func (f *Fetcher) FetchProtocols(ctx context.Context, dir string, period int) ([]string, error) {
	var paths []string
	for session := 1; ; session++ {
		path := ProtocolPath(dir, period, session)
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
			continue
		}
		err := f.download(ctx, f.ProtocolURL(period, session), path)
		if errors.Is(err, ErrNotFound) {
			return paths, nil
		}
		if err != nil {
			return paths, err
		}
		f.Logf("downloaded %s", path)
		paths = append(paths, path)
		if err := sleep(ctx, f.Delay); err != nil {
			return paths, err
		}
	}
}

// MdBPath is where the master data archive is cached below dir.
func MdBPath(dir string) string { return filepath.Join(dir, "MdB-Stammdaten.zip") }

// FetchMdB downloads the master data archive to dir unless it is cached or
// refresh is set, and returns its path.
func (f *Fetcher) FetchMdB(ctx context.Context, dir string, refresh bool) (string, error) {
	path := MdBPath(dir)
	if _, err := os.Stat(path); err == nil && !refresh {
		return path, nil
	}
	if err := f.download(ctx, f.MdBURL, path); err != nil {
		return "", err
	}
	f.Logf("downloaded %s", path)
	return path, nil
}

func (f *Fetcher) download(ctx context.Context, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := f.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("GET %s: %w", url, ErrNotFound)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Writing to a temporary file first means an interrupted download never
	// leaves a truncated file that later runs would treat as cached.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("GET %s: %w", url, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
