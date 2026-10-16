package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/stefankarlin/my-accessories/backend/internal/database"
)

// adminRoutes are mounted under /api/admin behind auth.RequireAdmin.
func (h *Handler) adminRoutes(r chi.Router) {
	r.Post("/exercises", h.CreateExercise)
	r.Get("/exercises", h.ListExercises)
	r.Get("/exercises/{id}", h.GetExercise)
	r.Put("/exercises/{id}", h.UpdateExercise)
	r.Get("/joints", h.ListJoints)
	r.Get("/muscle-groups", h.ListMuscleGroups)
	r.Post("/training-plans", h.CreateTrainingPlan)
	r.Post("/users/{userID}/trainings", h.AssignTraining)
}

type createExerciseRequest struct {
	Name           string      `json:"name"`
	JointIDs       []uuid.UUID `json:"joint_ids"`
	MuscleGroupIDs []uuid.UUID `json:"muscle_group_ids"`
	Description    string      `json:"description"`
	VideoURL       string      `json:"video_url"`
}

// parseExerciseRequest decodes and validates an exercise body, writing the error response
// itself on failure. The returned request has deduplicated ids.
func parseExerciseRequest(w http.ResponseWriter, r *http.Request) (createExerciseRequest, bool) {
	var req createExerciseRequest
	if !decodeJSON(w, r, &req) {
		return req, false
	}
	req.JointIDs, req.MuscleGroupIDs = unique(req.JointIDs), unique(req.MuscleGroupIDs)
	if req.Name == "" || len(req.JointIDs) == 0 || len(req.MuscleGroupIDs) == 0 {
		writeError(w, http.StatusBadRequest, "name, joint_ids and muscle_group_ids are required")
		return req, false
	}
	if req.VideoURL != "" {
		if u, err := url.Parse(req.VideoURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			writeError(w, http.StatusBadRequest, "video_url must be an http(s) URL")
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
		writeError(w, http.StatusNotFound, "exercise not found")
	case errors.As(err, &pgErr) && pgErr.Code == "23503": // foreign_key_violation
		writeError(w, http.StatusBadRequest, "unknown joint or muscle group")
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
		JointIds:       req.JointIDs,
		MuscleGroupIds: req.MuscleGroupIDs,
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
		writeError(w, http.StatusBadRequest, "invalid exercise id")
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
		JointIds:       req.JointIDs,
		MuscleGroupIds: req.MuscleGroupIDs,
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
		writeError(w, http.StatusBadRequest, "invalid exercise id")
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

// CreateTrainingPlan is not implemented yet.
func (h *Handler) CreateTrainingPlan(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w)
}

// AssignTraining assigns a training plan to a user. Not implemented yet.
func (h *Handler) AssignTraining(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w)
}
