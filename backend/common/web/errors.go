package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/auth"
	"github.com/erdira/homegym/backend/common/errs"
	"github.com/erdira/homegym/backend/common/log"
)

// errorBody is the JSON of every error response: {"error": "message"}.
type errorBody struct {
	Error string `json:"error"`
}

// WriteJSON sends v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status is already sent, so all that is left is to log it.
		slog.Error("encode response", "err", err)
	}
}

// WriteError sends a JSON error response and logs it.
func WriteError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	ClientError(w, r, status, msg, nil)
}

// ClientError is WriteError with the underlying cause added to the log (never to the response).
func ClientError(w http.ResponseWriter, r *http.Request, status int, msg string, err error) {
	log.RequestError(r, status, msg, err)
	WriteJSON(w, status, errorBody{Error: msg})
}

// ServerError answers 500 with a generic message; err is only logged.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	ClientError(w, r, http.StatusInternalServerError, "internal server error", err)
}

// errStatus maps the kinds of errs.Error to HTTP statuses.
var errStatus = map[errs.Kind]int{
	errs.KindInvalid:      http.StatusBadRequest,
	errs.KindNotFound:     http.StatusNotFound,
	errs.KindConflict:     http.StatusConflict,
	errs.KindUnauthorized: http.StatusUnauthorized,
}

// ServiceError answers the error a handler returned: an *errs.Error with its status and message,
// anything else with a 500. It is every module's strict ResponseErrorHandlerFunc.
func ServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var e *errs.Error
	if errors.As(err, &e) {
		if status, ok := errStatus[e.Kind]; ok {
			ClientError(w, r, status, e.Msg, e.Err)
			return
		}
	}
	ServerError(w, r, err)
}

// BodyError answers a request body that isn't valid JSON (or is too large). It is every module's
// strict RequestErrorHandlerFunc.
func BodyError(w http.ResponseWriter, r *http.Request, err error) {
	ClientError(w, r, http.StatusBadRequest, "invalid JSON body", err)
}

// MissingParam answers a required path or query parameter the request lacks.
func MissingParam(w http.ResponseWriter, r *http.Request, name string, err error) {
	ClientError(w, r, http.StatusBadRequest, name+" is required", err)
}

// InvalidParam answers a path or query parameter not of the spec's type, e.g. an id that isn't a
// UUID.
func InvalidParam(w http.ResponseWriter, r *http.Request, name string, err error) {
	ClientError(w, r, http.StatusBadRequest, "invalid "+name, err)
}

// CurrentUser returns the signed-in user's id, which RequireUser put into ctx.
func CurrentUser(ctx context.Context) (uuid.UUID, error) {
	id, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return uuid.Nil, errs.Unauthorized("unauthenticated")
	}
	return id, nil
}
