// Package logging configures the process-wide slog logger and provides an
// HTTP request logging middleware.
package logging

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Setup builds a logger from the given level ("debug", "info", "warn",
// "error"; default "info") and format ("text" or "json"; default "text"),
// installs it as the slog default (which also routes the standard log
// package through it) and returns it.
func Setup(level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	if level != "" {
		if err := lvl.UnmarshalText([]byte(level)); err != nil {
			return nil, fmt.Errorf("invalid LOG_LEVEL %q", level)
		}
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	switch strings.ToLower(format) {
	case "", "text":
		h = slog.NewTextHandler(os.Stderr, opts)
	case "json":
		h = slog.NewJSONHandler(os.Stderr, opts)
	default:
		return nil, fmt.Errorf("invalid LOG_FORMAT %q (want text or json)", format)
	}

	logger := slog.New(h)
	slog.SetDefault(logger)
	return logger, nil
}

// Middleware logs one line per request with method, path, status, response
// size, duration and the chi request ID (if middleware.RequestID runs first).
// 5xx responses are logged at error level, everything else at info.
func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				status := ww.Status()
				if status == 0 {
					status = http.StatusOK
				}
				level := slog.LevelInfo
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				logger.LogAttrs(r.Context(), level, "request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", middleware.GetReqID(r.Context())),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// RequestError logs why a request failed: msg is the message sent to the
// client, err the underlying cause (may be nil). 5xx statuses are logged at
// error level, everything else at warn.
func RequestError(r *http.Request, status int, msg string, err error) {
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	attrs := []slog.Attr{
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
		slog.String("reason", msg),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	}
	if err != nil {
		attrs = append(attrs, slog.Any("err", err))
	}
	slog.Default().LogAttrs(r.Context(), level, "request error", attrs...)
}
