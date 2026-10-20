package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/workout"
)

// GetWeek returns the week for the tracker: the active trainings, the sessions the chosen one
// (training, optional: else the first) plans for it and the workouts finished. The client sends its local week: the Monday as date (YYYY-MM-DD) and the bounds
// from and to (RFC 3339).
func (h *Handler) GetWeek(w http.ResponseWriter, r *http.Request) {
	// An unparseable date is left zero, which the service rejects; an unparseable training is
	// none, which picks the first.
	monday, _ := time.Parse(time.DateOnly, r.URL.Query().Get("date"))
	trainingID, _ := uuid.Parse(r.URL.Query().Get("training"))
	h.withPeriod(w, r, func(ctx context.Context, userID uuid.UUID, from, to time.Time) (any, error) {
		return h.workouts.Week(ctx, userID, trainingID, monday, from, to)
	})
}

// ListWorkouts returns the user's workouts finished in [from, to) (RFC 3339), newest first.
func (h *Handler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	h.withPeriod(w, r, func(ctx context.Context, userID uuid.UUID, from, to time.Time) (any, error) {
		return h.workouts.List(ctx, userID, from, to)
	})
}

// withPeriod parses the from and to query parameters and answers with what fetch returns.
// Unparseable times are left zero, which the service rejects.
func (h *Handler) withPeriod(w http.ResponseWriter, r *http.Request, fetch func(ctx context.Context, userID uuid.UUID, from, to time.Time) (any, error)) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	result, err := fetch(r.Context(), userID, from, to)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type startWorkoutRequest struct {
	WeekSessionID uuid.UUID `json:"week_session_id"`
}

// StartWorkout begins a workout of a planned session (a slot of the week plan): 201, or 200 with
// the workout already in progress of that slot.
func (h *Handler) StartWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var req startWorkoutRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	wo, created, err := h.workouts.Start(r.Context(), userID, req.WeekSessionID)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, wo)
}

// GetWorkout returns one of the user's workouts with its sets.
func (h *Handler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	wo, err := h.workouts.Get(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

type finishWorkoutRequest struct {
	Notes string `json:"notes"`
}

// FinishWorkout completes a workout with notes (or only updates the notes of a finished one).
func (h *Handler) FinishWorkout(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	var req finishWorkoutRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	wo, err := h.workouts.Finish(r.Context(), userID, id, req.Notes)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// DeleteWorkout discards a workout in progress or removes a finished one.
func (h *Handler) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	if err := h.workouts.Delete(r.Context(), userID, id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// LogSet stores a set as it was done, or corrects one logged before.
func (h *Handler) LogSet(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	var in workout.SetInput
	if !decodeJSON(w, r, &in) {
		return
	}
	set, err := h.workouts.LogSet(r.Context(), userID, id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, set)
}

// DeleteSet removes a set from a workout.
func (h *Handler) DeleteSet(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "workout")
	if !ok {
		return
	}
	setID, ok := uuidParam(w, r, "setID", "set")
	if !ok {
		return
	}
	if err := h.workouts.DeleteSet(r.Context(), userID, id, setID); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
