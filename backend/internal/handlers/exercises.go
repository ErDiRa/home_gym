package handlers

import (
	"net/http"

	"github.com/erdira/homegym/backend/internal/service/exercise"
	"github.com/erdira/homegym/backend/internal/service/feedback"
	"github.com/erdira/homegym/backend/internal/service/lookup"
	"github.com/erdira/homegym/backend/internal/service/maxes"
)

// ListExercises returns the user's exercises with the names of their links.
func (h *Handler) ListExercises(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.exercises.List)
}

// GetExercise returns one of the user's exercises with the ids of its links, for filling the
// exercise form.
func (h *Handler) GetExercise(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "exercise")
	if !ok {
		return
	}
	e, err := h.exercises.Get(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var in exercise.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.exercises.Create(r.Context(), userID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) UpdateExercise(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "exercise")
	if !ok {
		return
	}
	var in exercise.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.exercises.Update(r.Context(), userID, id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// RateExercise stores the user's rating (1-5) and notes for one of their exercises, replacing
// any earlier one.
func (h *Handler) RateExercise(w http.ResponseWriter, r *http.Request) {
	userID, exerciseID, ok := userAndID(w, r, "exercise")
	if !ok {
		return
	}
	var in feedback.RatingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	f, err := h.feedback.Rate(r.Context(), userID, exerciseID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// ListExerciseRatings returns how the user rated their exercises, for the exercise picker.
func (h *Handler) ListExerciseRatings(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.feedback.Ratings)
}

// AddExerciseMax adds a 1RM to the history of one of the user's exercises; the newest is the
// current max.
func (h *Handler) AddExerciseMax(w http.ResponseWriter, r *http.Request) {
	userID, exerciseID, ok := userAndID(w, r, "exercise")
	if !ok {
		return
	}
	var in maxes.MaxInput
	if !decodeJSON(w, r, &in) {
		return
	}
	m, err := h.maxes.Add(r.Context(), userID, exerciseID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// ListExerciseMaxes returns the user's 1RM history of all their exercises, newest first.
func (h *Handler) ListExerciseMaxes(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.maxes.List)
}

// DeleteExerciseMax removes an entry from the user's 1RM history.
func (h *Handler) DeleteExerciseMax(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "1RM")
	if !ok {
		return
	}
	if err := h.maxes.Delete(r.Context(), userID, id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ExerciseStats lists the exercises the user has done in workouts, most often first, with
// their rating of each.
func (h *Handler) ExerciseStats(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.feedback.Stats)
}

func (h *Handler) ListJoints(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.Joints)
}

func (h *Handler) ListMuscleGroups(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.MuscleGroups)
}

func (h *Handler) ListFocuses(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.Focuses)
}

func (h *Handler) CreateFocus(w http.ResponseWriter, r *http.Request) {
	var in lookup.FocusInput
	if !decodeJSON(w, r, &in) {
		return
	}
	focus, err := h.lookups.CreateFocus(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, focus)
}
