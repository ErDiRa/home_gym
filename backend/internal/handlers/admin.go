package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/database"
)

// adminRoutes are mounted under /api/admin behind auth.RequireAdmin.
func (h *Handler) adminRoutes(r chi.Router) {
	r.Post("/exercises", h.CreateExercise)
	r.Get("/exercises", h.ListExercises)
	r.Get("/exercises/{id}", h.GetExercise)
	r.Put("/exercises/{id}", h.UpdateExercise)
	r.Get("/joints", h.ListJoints)
	r.Get("/muscle-groups", h.ListMuscleGroups)
	r.Get("/focuses", h.ListFocuses)
	r.Post("/focuses", h.CreateFocus)
	r.Get("/block-kinds", h.ListBlockKinds)
	r.Get("/rep-units", h.ListRepUnits)
	r.Post("/rep-units", h.CreateRepUnit)
	r.Get("/load-units", h.ListLoadUnits)
	r.Post("/load-units", h.CreateLoadUnit)
	r.Get("/trainings", h.ListTrainings)
	r.Post("/trainings", h.CreateTraining)
	r.Get("/trainings/{id}", h.GetTraining)
	r.Put("/trainings/{id}", h.UpdateTraining)
	r.Delete("/trainings/{id}", h.DeleteTraining)
	r.Post("/trainings/{id}/duplicate", h.DuplicateTraining)
	r.Get("/clients", h.ListClients)
	r.Post("/clients", h.CreateClient)
	r.Get("/clients/{id}", h.GetClient)
	r.Put("/clients/{id}", h.UpdateClient)
	r.Delete("/clients/{id}", h.DeleteClient)
}

type createExerciseRequest struct {
	Name           string      `json:"name"`
	JointIDs       []uuid.UUID `json:"joint_ids"`
	MuscleGroupIDs []uuid.UUID `json:"muscle_group_ids"`
	FocusIDs       []uuid.UUID `json:"focus_ids"`
	Description    string      `json:"description"`
	VideoURL       string      `json:"video_url"`
	Type           string      `json:"type"`
}

// parseExerciseRequest decodes and validates an exercise body, writing the error response
// itself on failure. The returned request has deduplicated ids.
func parseExerciseRequest(w http.ResponseWriter, r *http.Request) (createExerciseRequest, bool) {
	var req createExerciseRequest
	if !decodeJSON(w, r, &req) {
		return req, false
	}
	req.JointIDs, req.MuscleGroupIDs, req.FocusIDs = unique(req.JointIDs), unique(req.MuscleGroupIDs), unique(req.FocusIDs)
	if req.Name == "" || len(req.JointIDs) == 0 || len(req.MuscleGroupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, "name, joint_ids and muscle_group_ids are required")
		return req, false
	}
	if req.Type != "mobility" && req.Type != "strength" {
		writeError(w, r, http.StatusBadRequest, "type must be mobility or strength")
		return req, false
	}
	if req.VideoURL != "" {
		if u, err := url.Parse(req.VideoURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			writeError(w, r, http.StatusBadRequest, "video_url must be an http(s) URL")
			return req, false
		}
	}
	return req, true
}

// exerciseWriteError maps a failed create/update to a response.
func exerciseWriteError(w http.ResponseWriter, r *http.Request, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		clientError(w, r, http.StatusNotFound, "exercise not found", err)
	case errors.As(err, &pgErr) && pgErr.Code == "23503": // foreign_key_violation
		clientError(w, r, http.StatusBadRequest, "unknown joint, muscle group or focus", err)
	default:
		serverError(w, r, err)
	}
}

func (h *Handler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	req, ok := parseExerciseRequest(w, r)
	if !ok {
		return
	}
	exercise, err := h.queries.CreateExercise(r.Context(), database.CreateExerciseParams{
		Name:           req.Name,
		Description:    req.Description,
		VideoUrl:       req.VideoURL,
		Type:           req.Type,
		JointIds:       req.JointIDs,
		MuscleGroupIds: req.MuscleGroupIDs,
		FocusIds:       req.FocusIDs,
	})
	if err != nil {
		exerciseWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, exercise)
}

func (h *Handler) UpdateExercise(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		clientError(w, r, http.StatusBadRequest, "invalid exercise id", err)
		return
	}
	req, ok := parseExerciseRequest(w, r)
	if !ok {
		return
	}
	exercise, err := h.queries.UpdateExercise(r.Context(), database.UpdateExerciseParams{
		ID:             id,
		Name:           req.Name,
		Description:    req.Description,
		VideoUrl:       req.VideoURL,
		Type:           req.Type,
		JointIds:       req.JointIDs,
		MuscleGroupIds: req.MuscleGroupIDs,
		FocusIds:       req.FocusIDs,
	})
	if err != nil {
		exerciseWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, exercise)
}

// GetExercise returns one exercise with the ids of its joints and muscle groups, for filling
// the exercise form. It is shared by the admin and client routes.
func (h *Handler) GetExercise(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		clientError(w, r, http.StatusBadRequest, "invalid exercise id", err)
		return
	}
	exercise, err := h.queries.GetExercise(r.Context(), id)
	if err != nil {
		exerciseWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, exercise)
}

// unique returns ids without duplicates, keeping the first occurrence of each.
func unique(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (h *Handler) ListJoints(w http.ResponseWriter, r *http.Request) {
	joints, err := h.queries.ListJoints(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, joints)
}

func (h *Handler) ListFocuses(w http.ResponseWriter, r *http.Request) {
	focuses, err := h.queries.ListFocuses(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, focuses)
}

type createFocusRequest struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

func (h *Handler) CreateFocus(w http.ResponseWriter, r *http.Request) {
	var req createFocusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	name, alias := strings.TrimSpace(req.Name), strings.TrimSpace(req.Alias)
	if name == "" || len(name) > 60 {
		writeError(w, r, http.StatusBadRequest, "name is required (max 60 characters)")
		return
	}
	if alias == "" || utf8.RuneCountInString(alias) > 10 {
		writeError(w, r, http.StatusBadRequest, "alias is required (max 10 characters)")
		return
	}
	focus, err := h.queries.CreateFocus(r.Context(), database.CreateFocusParams{Name: name, Alias: alias})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			writeError(w, r, http.StatusConflict, "this focus already exists")
			return
		}
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, focus)
}

func (h *Handler) ListMuscleGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.queries.ListMuscleGroups(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handler) ListExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.queries.ListExercises(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, exercises)
}
