package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/database"
)

// Clients are the rows of the users table, managed by admins.

func (h *Handler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.queries.ListUsers(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

type clientRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"` // only used when creating
}

// parseClientRequest decodes, normalizes and validates a client body, writing the error
// response itself on failure. The password is only checked when withPassword is set.
func parseClientRequest(w http.ResponseWriter, r *http.Request, withPassword bool) (clientRequest, bool) {
	var req clientRequest
	if !decodeJSON(w, r, &req) {
		return req, false
	}
	// Login lowercases the email it looks up, so store it the same way.
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FirstName, req.LastName = strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName)
	if addr, err := mail.ParseAddress(req.Email); err != nil || addr.Address != req.Email {
		writeError(w, r, http.StatusBadRequest, "a valid email is required")
		return req, false
	}
	if req.FirstName == "" || req.LastName == "" || tooLong(req.FirstName, 100) || tooLong(req.LastName, 100) {
		writeError(w, r, http.StatusBadRequest, "first_name and last_name are required (max 100 characters)")
		return req, false
	}
	// bcrypt only uses the first 72 bytes.
	if withPassword && (utf8.RuneCountInString(req.Password) < 8 || len(req.Password) > 72) {
		writeError(w, r, http.StatusBadRequest, "password must be 8 to 72 characters")
		return req, false
	}
	return req, true
}

// errEmailTaken means the email belongs to another client or to an admin.
var errEmailTaken = errors.New("email taken")

// checkNotAdminEmail rejects emails of admins: login checks users before admins, so a
// client with an admin's email would lock that admin out.
func (h *Handler) checkNotAdminEmail(r *http.Request, email string) error {
	_, err := h.queries.GetAdminByEmail(r.Context(), email)
	switch {
	case err == nil:
		return errEmailTaken
	case errors.Is(err, pgx.ErrNoRows):
		return nil
	default:
		return err
	}
}

// clientWriteError maps a failed client query to a response.
func clientWriteError(w http.ResponseWriter, r *http.Request, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, errEmailTaken), errors.As(err, &pgErr) && pgErr.Code == "23505": // unique_violation
		clientError(w, r, http.StatusConflict, "this email is already in use", err)
	case errors.Is(err, pgx.ErrNoRows):
		clientError(w, r, http.StatusNotFound, "client not found", err)
	default:
		serverError(w, r, err)
	}
}

func (h *Handler) CreateClient(w http.ResponseWriter, r *http.Request) {
	req, ok := parseClientRequest(w, r, true)
	if !ok {
		return
	}
	if err := h.checkNotAdminEmail(r, req.Email); err != nil {
		clientWriteError(w, r, err)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		serverError(w, r, err)
		return
	}
	client, err := h.queries.CreateUser(r.Context(), database.CreateUserParams{
		Email:        req.Email,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PasswordHash: string(hash),
	})
	if err != nil {
		clientWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, client)
}

// UpdateClient changes a client's name and email. The password can't be changed here.
func (h *Handler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id, ok := clientIDParam(w, r)
	if !ok {
		return
	}
	req, ok := parseClientRequest(w, r, false)
	if !ok {
		return
	}
	if err := h.checkNotAdminEmail(r, req.Email); err != nil {
		clientWriteError(w, r, err)
		return
	}
	client, err := h.queries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:        id,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		clientWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, client)
}

type clientResponse struct {
	database.GetUserByIDRow
	Trainings []database.ListClientTrainingsRow `json:"trainings"`
}

// GetClient returns a client with their trainings, the active one first.
func (h *Handler) GetClient(w http.ResponseWriter, r *http.Request) {
	id, ok := clientIDParam(w, r)
	if !ok {
		return
	}
	client, err := h.queries.GetUserByID(r.Context(), id)
	if err != nil {
		clientWriteError(w, r, err)
		return
	}
	trainings, err := h.queries.ListClientTrainings(r.Context(), id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, clientResponse{GetUserByIDRow: client, Trainings: orEmpty(trainings)})
}

// DeleteClient soft-deletes a client; they can no longer sign in, their open sessions are
// rejected by auth.RequireUser, and their trainings are hidden.
func (h *Handler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id, ok := clientIDParam(w, r)
	if !ok {
		return
	}
	n, err := h.queries.DeleteUser(r.Context(), id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if n == 0 {
		writeError(w, r, http.StatusNotFound, "client not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func clientIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		clientError(w, r, http.StatusBadRequest, "invalid client id", err)
	}
	return id, err == nil
}
