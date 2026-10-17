package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/logging"
	"github.com/erdira/homegym/backend/internal/service"
)

// writeJSON sends v as a JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status is already sent, so all that is left is to log it.
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

// serverError answers 500 with a generic message; err is only logged.
func serverError(w http.ResponseWriter, r *http.Request, err error) {
	clientError(w, r, http.StatusInternalServerError, "internal server error", err)
}

// serviceStatus maps the kinds of service errors to HTTP statuses.
var serviceStatus = map[service.Kind]int{
	service.KindInvalid:      http.StatusBadRequest,
	service.KindNotFound:     http.StatusNotFound,
	service.KindConflict:     http.StatusConflict,
	service.KindUnauthorized: http.StatusUnauthorized,
}

// serviceError answers a failed service call: a *service.Error with its status and message,
// anything else with a 500.
func serviceError(w http.ResponseWriter, r *http.Request, err error) {
	var svcErr *service.Error
	if errors.As(err, &svcErr) {
		if status, ok := serviceStatus[svcErr.Kind]; ok {
			clientError(w, r, status, svcErr.Msg, svcErr.Err)
			return
		}
	}
	serverError(w, r, err)
}

// decodeJSON decodes the request body (max 1 MiB) into dst. On failure it has already written
// the 400 response, so callers just return.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		clientError(w, r, http.StatusBadRequest, "invalid JSON body", err)
		return false
	}
	return true
}

// idParam parses the {id} URL parameter, writing a 400 ("invalid <what> id") when it isn't a
// UUID (callers just return when ok is false).
func idParam(w http.ResponseWriter, r *http.Request, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		clientError(w, r, http.StatusBadRequest, "invalid "+what+" id", err)
	}
	return id, err == nil
}

// userIDFromRequest returns the signed-in account's ID set by auth.Middleware, writing a 401
// when it is missing (callers just return when ok is false).
func userIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthenticated")
	}
	return id, ok
}
