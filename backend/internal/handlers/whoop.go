package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/logging"
)

// whoopStateCookie carries the OAuth state from ConnectWhoop to WhoopCallback, so the callback
// only accepts a code for the connection this browser started.
const whoopStateCookie = "whoop_state"

// settingsPage is where the browser returns to after connecting WHOOP.
const settingsPage = "/track/settings"

// WhoopStatus tells whether WHOOP can be connected (the app is configured) and whether it is.
func (h *Handler) WhoopStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	connected, err := h.whoop.Connected(r.Context(), userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"configured": h.whoop.Configured(), "connected": connected})
}

// ConnectWhoop sends the browser to WHOOP to allow HomeGym access; WHOOP returns it to
// WhoopCallback.
func (h *Handler) ConnectWhoop(w http.ResponseWriter, r *http.Request) {
	if !h.whoop.Configured() {
		writeError(w, r, http.StatusNotFound, "WHOOP is not configured")
		return
	}
	b := make([]byte, 8)
	rand.Read(b)
	state := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name: whoopStateCookie, Value: state, Path: "/api/whoop", MaxAge: 600,
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.whoop.AuthURL(state), http.StatusFound)
}

// WhoopCallback completes the connection with the code WHOOP sent and returns the browser to
// the settings page (with ?whoop=error when it failed; the cause is logged).
func (h *Handler) WhoopCallback(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: whoopStateCookie, Path: "/api/whoop", MaxAge: -1, HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode})
	q := r.URL.Query()
	cookie, err := r.Cookie(whoopStateCookie)
	switch {
	case err != nil || q.Get("state") == "" || cookie.Value != q.Get("state"):
		logging.RequestError(r, http.StatusBadRequest, "WHOOP callback with an unknown state", err)
	case q.Get("error") != "" || q.Get("code") == "":
		logging.RequestError(r, http.StatusBadRequest, "WHOOP declined the connection: "+q.Get("error"), nil)
	default:
		if err := h.whoop.Connect(r.Context(), userID, q.Get("code")); err != nil {
			logging.RequestError(r, http.StatusBadGateway, "connect WHOOP", err)
		} else {
			http.Redirect(w, r, settingsPage, http.StatusFound)
			return
		}
	}
	http.Redirect(w, r, settingsPage+"?whoop=error", http.StatusFound)
}

// DisconnectWhoop forgets the user's WHOOP tokens.
func (h *Handler) DisconnectWhoop(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	if err := h.whoop.Disconnect(r.Context(), userID); err != nil {
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListWhoopCandidates lists the WHOOP activities of the workout's day ([from, to), RFC 3339,
// the client's local day) to pick the one to link.
func (h *Handler) ListWhoopCandidates(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "workout")
	if !ok {
		return
	}
	h.withPeriod(w, r, func(ctx context.Context, userID uuid.UUID, from, to time.Time) (any, error) {
		return h.whoop.Candidates(ctx, userID, id, from, to)
	})
}

// GetWorkoutWhoop returns the WHOOP data linked to a workout, or null.
func (h *Handler) GetWorkoutWhoop(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	d, err := h.whoop.Get(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

type linkWhoopRequest struct {
	WhoopWorkoutID uuid.UUID `json:"whoop_workout_id"`
}

// LinkWhoop links a WHOOP activity to a workout, fetching its data (again).
func (h *Handler) LinkWhoop(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	var req linkWhoopRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	d, err := h.whoop.Link(r.Context(), userID, id, req.WhoopWorkoutID)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// UnlinkWhoop removes the WHOOP data from a workout.
func (h *Handler) UnlinkWhoop(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	if err := h.whoop.Unlink(r.Context(), userID, id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
