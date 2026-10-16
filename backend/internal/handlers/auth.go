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
	"github.com/stefankarlin/my-accessories/backend/internal/database"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type userResponse struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	IsAdmin bool      `json:"is_admin"`
}

// dummyHash lets Login spend the same time on unknown emails as on wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	account, role, ok := h.authenticate(w, r, email, req.Password)
	if !ok {
		return
	}
	h.setSession(w, r, role, account, time.Now(), req.Remember)
}

// authenticate checks the credentials against the users table, then the admins table,
// and writes the error response itself on failure.
func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request, email, password string) (userResponse, string, bool) {
	var (
		account userResponse
		role    string
		hash    string
	)
	user, err := h.queries.GetUserByEmail(r.Context(), email)
	switch {
	case err == nil:
		account, role, hash = userResponse{ID: user.ID, Email: user.Email}, auth.RoleUser, user.PasswordHash
	case errors.Is(err, pgx.ErrNoRows):
		admin, err := h.queries.GetAdminByEmail(r.Context(), email)
		switch {
		case err == nil:
			account, role, hash = userResponse{ID: admin.ID, Email: admin.Email, IsAdmin: true}, auth.RoleAdmin, admin.PasswordHash
		case errors.Is(err, pgx.ErrNoRows):
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return userResponse{}, "", false
		default:
			serverError(w, r, err)
			return userResponse{}, "", false
		}
	default:
		serverError(w, r, err)
		return userResponse{}, "", false
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return userResponse{}, "", false
	}
	return account, role, true
}

// setSession issues a fresh token and cookie for user and writes the user as the response.
// The token lasts auth.SessionTTL but never beyond auth.MaxSessionAge after authTime. With
// "remember me" the cookie expires with the token; otherwise it is dropped when the browser closes.
func (h *Handler) setSession(w http.ResponseWriter, r *http.Request, role string, user userResponse, authTime time.Time, remember bool) {
	now := time.Now()
	expiry := auth.SessionExpiry(now, authTime)
	token, err := h.tokens.Issue(role, user.ID, authTime, remember, expiry)
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
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id, err := claims.UserID()
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var account userResponse
	if claims.Role == auth.RoleAdmin {
		var admin database.GetAdminByIDRow
		admin, err = h.queries.GetAdminByID(r.Context(), id)
		account = userResponse{ID: admin.ID, Email: admin.Email, IsAdmin: true}
	} else {
		var user database.GetUserByIDRow
		user, err = h.queries.GetUserByID(r.Context(), id)
		account = userResponse{ID: user.ID, Email: user.Email}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		auth.ClearCookie(w, h.secureCookies)
		writeError(w, http.StatusUnauthorized, "account no longer exists")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	h.setSession(w, r, claims.Role, account, time.Unix(claims.AuthTime, 0), claims.Remember)
}
