package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/stefankarlin/my-accessories/backend/internal/auth"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// dummyHash lets Login spend the same time on unknown emails as on wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.queries.GetUserByEmail(r.Context(), email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	h.setSession(w, r, userResponse{ID: user.ID, Email: user.Email}, time.Now(), req.Remember)
}

// setSession issues a fresh token and cookie for user and writes the user as the response.
// The token lasts auth.SessionTTL but never beyond auth.MaxSessionAge after authTime. With
// "remember me" the cookie expires with the token; otherwise it is dropped when the browser closes.
func (h *Handler) setSession(w http.ResponseWriter, r *http.Request, user userResponse, authTime time.Time, remember bool) {
	now := time.Now()
	expiry := auth.SessionExpiry(now, authTime)
	token, err := h.tokens.Issue(user.ID, authTime, remember, expiry)
	if err != nil {
		serverError(w, r, err)
		return
	}
	var cookieTTL time.Duration
	if remember {
		cookieTTL = expiry.Sub(now)
	}
	auth.SetCookie(w, token, cookieTTL, h.secureCookies)
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	auth.ClearCookie(w, h.secureCookies)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user and extends the session (up to auth.MaxSessionAge
// after login). The frontend uses it to restore the session since it cannot read the
// HttpOnly cookie.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	user, err := h.queries.GetUserByID(r.Context(), userID)
	if errors.Is(err, pgx.ErrNoRows) {
		auth.ClearCookie(w, h.secureCookies)
		writeError(w, http.StatusUnauthorized, "user no longer exists")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	h.setSession(w, r, userResponse{ID: user.ID, Email: user.Email}, time.Unix(claims.AuthTime, 0), claims.Remember)
}
