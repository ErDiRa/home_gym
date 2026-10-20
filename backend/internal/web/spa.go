// Package web serves the built frontend (frontend/app/dist) next to the API.
package web

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	// Vite puts a content hash into every file name under assets/, so they never change.
	cacheImmutable = "public, max-age=31536000, immutable"
	// index.html names the current assets, so the browser must revalidate it on every load.
	cacheRevalidate = "no-cache"
	// Other files (e.g. from public/) keep their names across builds.
	cacheShort = "public, max-age=3600"
)

// SPA serves the files in dir. Paths that match no file and have no extension are client-side
// routes (e.g. /track/history) and get index.html; a missing file with an extension (an asset
// of an older build) is a 404, so the browser never gets HTML for a script.
func SPA(dir string) http.Handler {
	fsys := os.DirFS(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if fs.ValidPath(name) && isFile(fsys, name) {
			w.Header().Set("Cache-Control", cacheControl(name))
			http.ServeFileFS(w, r, fsys, name)
			return
		}
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", cacheRevalidate)
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}

func isFile(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && info.Mode().IsRegular()
}

func cacheControl(name string) string {
	switch {
	case name == "index.html":
		return cacheRevalidate
	case strings.HasPrefix(name, "assets/"):
		return cacheImmutable
	default:
		return cacheShort
	}
}
