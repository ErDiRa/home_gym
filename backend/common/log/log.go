// Package logging configures the process-wide slog logger and provides an
// HTTP request logging middleware.
package log

import (
	"context"
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

// failureKey is the context key of the *failure that Middleware leaves for RequestError.
type failureKey struct{}

// failure is why a request failed, filled in by RequestError and logged by Middleware.
type failure struct {
	msg string
	err error
}

// Middleware logs one line per request with method, path, status, response
// size, duration and the chi request ID (if middleware.RequestID runs first).
// Successful requests are logged at debug level, so info stays quiet; 4xx at
// warn and 5xx at error, with the reason and cause from RequestError.
func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the writer to read the status and size the handler wrote.
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			f := &failure{}

			defer func() {
				status := ww.Status()
				// No explicit WriteHeader (or nothing written at all) means net/http sent 200.
				if status == 0 {
					status = http.StatusOK
				}
				attrs := []slog.Attr{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", middleware.GetReqID(r.Context())),
				}
				if f.msg != "" {
					attrs = append(attrs, slog.String("reason", f.msg))
				}
				if f.err != nil {
					attrs = append(attrs, slog.Any("err", f.err))
				}
				logger.LogAttrs(r.Context(), statusLevel(status), "request", attrs...)
			}()

			next.ServeHTTP(ww, r.WithContext(context.WithValue(r.Context(), failureKey{}, f)))
		})
	}
}

func statusLevel(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}

// RequestError records why a request failed: msg is the message sent to the
// client, err the underlying cause (may be nil). Middleware adds both to its
// request line; outside of it the failure is logged on its own, at error level
// for 5xx and warn for everything else.
func RequestError(r *http.Request, status int, msg string, err error) {
	if record(r.Context(), msg, err) {
		return
	}
	logFailure(r.Context(), status, msg, err, slog.String("method", r.Method), slog.String("path", r.URL.Path))
}

// ContextError is RequestError for code that has the request's context but not the request,
// such as the generated API handlers.
func ContextError(ctx context.Context, status int, msg string, err error) {
	if record(ctx, msg, err) {
		return
	}
	logFailure(ctx, status, msg, err)
}

// record hands the failure to Middleware's request line, reporting whether it runs.
func record(ctx context.Context, msg string, err error) bool {
	f, ok := ctx.Value(failureKey{}).(*failure)
	if ok {
		f.msg, f.err = msg, err
	}
	return ok
}

// logFailure logs a failure outside Middleware as its own line.
func logFailure(ctx context.Context, status int, msg string, err error, attrs ...slog.Attr) {
	attrs = append(attrs,
		slog.Int("status", status),
		slog.String("reason", msg),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)
	if err != nil {
		attrs = append(attrs, slog.Any("err", err))
	}
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	slog.Default().LogAttrs(ctx, level, "request error", attrs...)
}
