package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPA(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "<html>app</html>")
	write("assets/app-abc123.js", "console.log(1)")
	write("logo.svg", "<svg/>")
	h := SPA(dir)

	tests := []struct {
		name, method, path string
		status             int
		body, cache        string
	}{
		{"root", http.MethodGet, "/", http.StatusOK, "<html>app</html>", cacheRevalidate},
		{"hashed asset", http.MethodGet, "/assets/app-abc123.js", http.StatusOK, "console.log(1)", cacheImmutable},
		{"public file", http.MethodGet, "/logo.svg", http.StatusOK, "<svg/>", cacheShort},
		{"client route", http.MethodGet, "/track/workouts/5", http.StatusOK, "<html>app</html>", cacheRevalidate},
		{"directory is a client route", http.MethodGet, "/assets", http.StatusOK, "<html>app</html>", cacheRevalidate},
		{"missing asset", http.MethodGet, "/assets/app-old.js", http.StatusNotFound, "", ""},
		{"dot-dot is rejected", http.MethodGet, "/../../etc/passwd", http.StatusBadRequest, "", ""},
		{"head", http.MethodHead, "/", http.StatusOK, "", cacheRevalidate},
		{"post", http.MethodPost, "/", http.StatusMethodNotAllowed, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			req.URL.Path = tt.path // set directly so ".." isn't resolved by the request parser
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.cache {
				t.Errorf("Cache-Control = %q, want %q", got, tt.cache)
			}
			if tt.body != "" && rec.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.body)
			}
		})
	}
}
