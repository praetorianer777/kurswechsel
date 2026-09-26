package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHandler(t *testing.T) {
	h := Handler(fstest.MapFS{
		"index.html":      {Data: []byte("<html>app</html>")},
		"assets/app-1.js": {Data: []byte("js")},
		"favicon.svg":     {Data: []byte("<svg/>")},
	})
	for _, tc := range []struct {
		path, body, cache string
		code              int
	}{
		{path: "/", body: "<html>app</html>", code: 200, cache: "no-cache"},
		{path: "/person/11004000", body: "<html>app</html>", code: 200, cache: "no-cache"},
		{path: "/assets/app-1.js", body: "js", code: 200, cache: "public, max-age=31536000, immutable"},
		{path: "/favicon.svg", body: "<svg/>", code: 200},
		{path: "/missing.png", code: 404},
	} {
		rec := get(h, tc.path)
		if rec.Code != tc.code {
			t.Errorf("%s: status %d, want %d", tc.path, rec.Code, tc.code)
		}
		if tc.body != "" && rec.Body.String() != tc.body {
			t.Errorf("%s: body %q", tc.path, rec.Body.String())
		}
		if tc.cache != "" && rec.Header().Get("Cache-Control") != tc.cache {
			t.Errorf("%s: Cache-Control %q", tc.path, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestHandlerWithoutBuild(t *testing.T) {
	rec := get(Handler(fstest.MapFS{".keep": {}}), "/")
	if rec.Code != 404 {
		t.Errorf("status = %d", rec.Code)
	}
	if Dist() == nil {
		t.Error("Dist() = nil")
	}
}
