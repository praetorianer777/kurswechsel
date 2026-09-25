package corpus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// UserAgent identifies the project to the Bundestag's servers.
const UserAgent = "kurswechsel-spike/0.1 (+https://github.com/praetorianer777/kurswechsel)"

// ProtocolURL is where the Bundestag publishes the XML of one session.
func ProtocolURL(period, session int) string {
	return fmt.Sprintf("https://dserver.bundestag.de/btp/%d/%d%03d.xml", period, period, session)
}

// Fetch downloads the protocols of one period into dir/<period>/, skipping
// files it already has. Sessions are numbered without gaps, so the first
// missing session ends the period.
func Fetch(ctx context.Context, client *http.Client, dir string, period int, logf func(string, ...any)) (int, error) {
	target := filepath.Join(dir, fmt.Sprint(period))
	if err := os.MkdirAll(target, 0o755); err != nil {
		return 0, err
	}
	n := 0
	for session := 1; ; session++ {
		path := filepath.Join(target, fmt.Sprintf("%d%03d.xml", period, session))
		if _, err := os.Stat(path); err == nil {
			n++
			continue
		}
		ok, err := download(ctx, client, ProtocolURL(period, session), path)
		if err != nil {
			return n, err
		}
		if !ok {
			return n, nil
		}
		n++
		logf("fetched %s", path)
		select {
		case <-ctx.Done():
			return n, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func download(ctx context.Context, client *http.Client, url, path string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return false, err
	}
	if err := f.Close(); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, path)
}
