package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/internal/api"
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
	writeJSON(w, status, api.Error{Error: msg})
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

// serviceError answers the error a handler returned: a *service.Error with its status and
// message, anything else with a 500.
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

// bodyError answers a request body that isn't valid JSON (or is too large).
func bodyError(w http.ResponseWriter, r *http.Request, err error) {
	clientError(w, r, http.StatusBadRequest, "invalid JSON body", err)
}

// paramError answers a path or query parameter that is missing or not of the spec's type,
// e.g. an id that isn't a UUID.
func paramError(w http.ResponseWriter, r *http.Request, err error) {
	var required *api.RequiredParamError
	var invalid *api.InvalidParamFormatError
	switch {
	case errors.As(err, &required):
		clientError(w, r, http.StatusBadRequest, required.ParamName+" is required", err)
	case errors.As(err, &invalid):
		clientError(w, r, http.StatusBadRequest, "invalid "+invalid.ParamName, err)
	default:
		clientError(w, r, http.StatusBadRequest, "invalid request", err)
	}
}

// Body limits: 1 MiB, except a cycle import (a 10-week cycle with ids is a few hundred KB).
const (
	maxBody       = 1 << 20
	maxImportBody = 8 << 20
)

// limitBody caps the request body; a larger one fails decoding (400).
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxBody)
		if r.URL.Path == "/api/cycles/import" {
			limit = maxImportBody
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

// currentUser returns the signed-in user's ID, which the auth middleware put into ctx.
func currentUser(ctx context.Context) (uuid.UUID, error) {
	id, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return uuid.Nil, service.Unauthorized("unauthenticated")
	}
	return id, nil
}

// listOwn returns the rows fetch returns for the signed-in user, mapped with toAPI.
func listOwn[T, U any](ctx context.Context, fetch func(context.Context, uuid.UUID) ([]T, error), toAPI func(T) U) ([]U, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := fetch(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapSlice(rows, toAPI), nil
}

// list returns the rows fetch returns, mapped with toAPI.
func list[T, U any](ctx context.Context, fetch func(context.Context) ([]T, error), toAPI func(T) U) ([]U, error) {
	rows, err := fetch(ctx)
	if err != nil {
		return nil, err
	}
	return mapSlice(rows, toAPI), nil
}

// The mappers between the service types and the API models are named <thing>ToAPI and
// <thing>FromAPI and live next to the handlers that use them.

// mapSlice maps every element with f; the result is never nil, so it encodes as [].
func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// date is a database date as an API date.
func date(d pgtype.Date) openapi_types.Date {
	return openapi_types.Date{Time: d.Time}
}

// timestamp is a NOT NULL database timestamp as an API date-time.
func timestamp(t pgtype.Timestamptz) time.Time {
	return t.Time
}

// optionalTimestamp is a nullable database timestamp as an API date-time, nil when NULL.
func optionalTimestamp(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// value returns *p, or the zero value when p is nil (an optional field the client left out).
func value[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
