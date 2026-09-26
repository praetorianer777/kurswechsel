// Package web serves the built frontend, which is embedded into the binary.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// The frontend build copies its output into dist/ before the Go build;
// without it only .keep is embedded and Handler answers with a hint.
//
//go:embed all:dist
var embedded embed.FS

// Dist is the embedded build output.
func Dist() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}

// Handler serves static files from fsys. Paths without a file extension that
// do not exist are client-side routes and get index.html.
func Handler(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" || p == "index.html" {
			serveIndex(w, r, fsys)
			return
		}
		if _, err := fs.Stat(fsys, p); err != nil {
			if path.Ext(p) != "" {
				http.NotFound(w, r)
				return
			}
			serveIndex(w, r, fsys)
			return
		}
		if strings.HasPrefix(p, "assets/") {
			// Vite puts a content hash into every asset name.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	b, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "frontend not built: run npm run build in frontend/", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if r.Method != http.MethodHead {
		w.Write(b)
	}
}
