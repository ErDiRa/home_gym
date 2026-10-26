package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    map[string]any // fields of the one logged line; nil = nothing logged at info
	}{
		{
			name:    "success is debug only",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		},
		{
			name: "client error carries the reason",
			handler: func(w http.ResponseWriter, r *http.Request) {
				RequestError(r, http.StatusBadRequest, "invalid JSON body", errors.New("unexpected EOF"))
				w.WriteHeader(http.StatusBadRequest)
			},
			want: map[string]any{"level": "WARN", "status": 400.0, "reason": "invalid JSON body", "err": "unexpected EOF"},
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				RequestError(r, http.StatusInternalServerError, "internal server error", errors.New("db down"))
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: map[string]any{"level": "ERROR", "status": 500.0, "reason": "internal server error", "err": "db down"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

			Middleware(logger)(tt.handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/x", nil))

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if tt.want == nil {
				if buf.Len() != 0 {
					t.Fatalf("logged %q, want nothing", buf.String())
				}
				return
			}
			if len(lines) != 1 {
				t.Fatalf("logged %d lines, want 1: %q", len(lines), buf.String())
			}
			var got map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
