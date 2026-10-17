package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func notImplemented(w http.ResponseWriter) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func userIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
	}
	return id, ok
}
