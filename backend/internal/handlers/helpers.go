package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/logging"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// writeError sends a JSON error response and logs it.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	clientError(w, r, status, msg, nil)
}

// clientError is writeError with the underlying cause added to the log (never to the response).
func clientError(w http.ResponseWriter, r *http.Request, status int, msg string, err error) {
	logging.RequestError(r, status, msg, err)
	writeJSON(w, status, map[string]string{"error": msg})
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotImplemented, "not implemented")
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	clientError(w, r, http.StatusInternalServerError, "internal server error", err)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		clientError(w, r, http.StatusBadRequest, "invalid JSON body", err)
		return false
	}
	return true
}

func userIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthenticated")
	}
	return id, ok
}
