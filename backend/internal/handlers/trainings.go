package handlers

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/database"
)

// A training belongs to one client and is a tree: units (e.g. "Day A", done on weekdays
// and/or dates) hold blocks (e.g. "Warm-up", "Superset"), and blocks hold one or more
// exercises with their prescription. The same shape is sent to and returned from the API.

type trainingExerciseRequest struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Sets       int32     `json:"sets"`
	Reps       string    `json:"reps"`
	Notes      string    `json:"notes"`
}

type trainingBlockRequest struct {
	Name      string                    `json:"name"`
	Notes     string                    `json:"notes"`
	Exercises []trainingExerciseRequest `json:"exercises"`
}

type trainingUnitRequest struct {
	// ID is set for units that already exist, so they keep their id; new units have none.
	ID       *uuid.UUID             `json:"id"`
	Name     string                 `json:"name"`
	Notes    string                 `json:"notes"`
	Weekdays []int16                `json:"weekdays"` // ISO, 1 = Monday
	Dates    []string               `json:"dates"`    // YYYY-MM-DD
	Blocks   []trainingBlockRequest `json:"blocks"`

	dates []pgtype.Date // Dates, parsed by parseTrainingRequest
}

type trainingRequest struct {
	ClientID    uuid.UUID             `json:"client_id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Active      bool                  `json:"active"`
	Units       []trainingUnitRequest `json:"units"`
}

var (
	// errUnknownUnit means a unit id in the request doesn't belong to the training.
	errUnknownUnit = errors.New("unknown unit")
	// errUnknownClient means the training's client doesn't exist or was deleted.
	errUnknownClient = errors.New("unknown client")
)

func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// parseTrainingRequest decodes, trims and validates a training body, writing the error
// response itself on failure.
func parseTrainingRequest(w http.ResponseWriter, r *http.Request) (trainingRequest, bool) {
	var req trainingRequest
	if !decodeJSON(w, r, &req) {
		return req, false
	}
	fail := func(msg string) (trainingRequest, bool) {
		writeError(w, http.StatusBadRequest, msg)
		return req, false
	}

	req.Name, req.Description = strings.TrimSpace(req.Name), strings.TrimSpace(req.Description)
	if req.ClientID == uuid.Nil {
		return fail("client_id is required")
	}
	if req.Name == "" || tooLong(req.Name, 100) {
		return fail("name is required (max 100 characters)")
	}
	seenUnits := map[uuid.UUID]bool{}
	for i := range req.Units {
		u := &req.Units[i]
		u.Name, u.Notes = strings.TrimSpace(u.Name), strings.TrimSpace(u.Notes)
		if u.Name == "" || tooLong(u.Name, 100) {
			return fail("every unit needs a name (max 100 characters)")
		}
		if u.ID != nil {
			if seenUnits[*u.ID] {
				return fail("a unit appears twice")
			}
			seenUnits[*u.ID] = true
		}
		weekdays, ok := normalizeWeekdays(u.Weekdays)
		if !ok {
			return fail("weekdays must be 1 (Monday) to 7 (Sunday)")
		}
		u.Weekdays = weekdays
		if u.dates, ok = parseDates(u.Dates); !ok {
			return fail("dates must be formatted YYYY-MM-DD")
		}
		for j := range u.Blocks {
			b := &u.Blocks[j]
			b.Name, b.Notes = strings.TrimSpace(b.Name), strings.TrimSpace(b.Notes)
			if tooLong(b.Name, 100) {
				return fail("block names can have at most 100 characters")
			}
			if len(b.Exercises) == 0 {
				return fail("every block needs at least one exercise")
			}
			for k := range b.Exercises {
				e := &b.Exercises[k]
				e.Reps, e.Notes = strings.TrimSpace(e.Reps), strings.TrimSpace(e.Notes)
				if e.ExerciseID == uuid.Nil || e.Sets < 1 || e.Sets > 100 || e.Reps == "" || tooLong(e.Reps, 30) {
					return fail("every exercise needs an exercise_id, sets (1-100) and reps (max 30 characters)")
				}
			}
		}
	}
	return req, true
}

// normalizeWeekdays sorts and deduplicates ISO weekdays, reporting false for invalid ones.
func normalizeWeekdays(days []int16) ([]int16, bool) {
	var seen [8]bool
	for _, d := range days {
		if d < 1 || d > 7 {
			return nil, false
		}
		seen[d] = true
	}
	out := []int16{}
	for d := int16(1); d <= 7; d++ {
		if seen[d] {
			out = append(out, d)
		}
	}
	return out, true
}

// parseDates parses YYYY-MM-DD dates, sorted and deduplicated, reporting false for invalid ones.
func parseDates(raw []string) ([]pgtype.Date, bool) {
	parsed := make([]time.Time, 0, len(raw))
	for _, r := range raw {
		d, err := time.Parse(time.DateOnly, r)
		if err != nil {
			return nil, false
		}
		parsed = append(parsed, d)
	}
	slices.SortFunc(parsed, func(a, b time.Time) int { return a.Compare(b) })
	parsed = slices.Compact(parsed)
	out := make([]pgtype.Date, 0, len(parsed))
	for _, d := range parsed {
		out = append(out, pgtype.Date{Time: d, Valid: true})
	}
	return out, true
}

// saveTrainingUnits writes the units, blocks and exercises of a training. Existing units are
// updated in place and units missing from the request are deleted; blocks and exercises
// are rewritten.
func saveTrainingUnits(ctx context.Context, q *database.Queries, trainingID uuid.UUID, units []trainingUnitRequest) error {
	keep := []uuid.UUID{}
	for _, u := range units {
		if u.ID != nil {
			keep = append(keep, *u.ID)
		}
	}
	if err := q.DeleteTrainingUnitsExcept(ctx, database.DeleteTrainingUnitsExceptParams{TrainingID: trainingID, KeepIds: keep}); err != nil {
		return err
	}
	if err := q.DeleteTrainingBlocks(ctx, trainingID); err != nil {
		return err
	}

	for i, u := range units {
		position := int32(i + 1)
		var unitID uuid.UUID
		if u.ID != nil {
			n, err := q.UpdateTrainingUnit(ctx, database.UpdateTrainingUnitParams{
				ID: *u.ID, TrainingID: trainingID, Position: position, Name: u.Name, Notes: u.Notes,
				Weekdays: u.Weekdays, Dates: u.dates,
			})
			if err != nil {
				return err
			}
			if n == 0 {
				return errUnknownUnit
			}
			unitID = *u.ID
		} else {
			id, err := q.InsertTrainingUnit(ctx, database.InsertTrainingUnitParams{
				TrainingID: trainingID, Position: position, Name: u.Name, Notes: u.Notes,
				Weekdays: u.Weekdays, Dates: u.dates,
			})
			if err != nil {
				return err
			}
			unitID = id
		}

		for j, b := range u.Blocks {
			blockID, err := q.InsertTrainingBlock(ctx, database.InsertTrainingBlockParams{
				UnitID: unitID, Position: int32(j + 1), Name: b.Name, Notes: b.Notes,
			})
			if err != nil {
				return err
			}
			params := database.InsertBlockExercisesParams{BlockID: blockID}
			for _, e := range b.Exercises {
				params.ExerciseIds = append(params.ExerciseIds, e.ExerciseID)
				params.Sets = append(params.Sets, e.Sets)
				params.Reps = append(params.Reps, e.Reps)
				params.Notes = append(params.Notes, e.Notes)
			}
			if err := q.InsertBlockExercises(ctx, params); err != nil {
				return err
			}
		}
	}
	return nil
}

// trainingWriteError maps a failed training query to a response.
func trainingWriteError(w http.ResponseWriter, r *http.Request, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "training not found")
	case errors.Is(err, errUnknownUnit):
		writeError(w, http.StatusBadRequest, "unknown unit")
	case errors.Is(err, errUnknownClient):
		writeError(w, http.StatusBadRequest, "unknown client")
	case errors.As(err, &pgErr) && pgErr.Code == "23503": // foreign_key_violation
		writeError(w, http.StatusBadRequest, "unknown exercise")
	default:
		serverError(w, r, err)
	}
}

func (h *Handler) ListTrainings(w http.ResponseWriter, r *http.Request) {
	trainings, err := h.queries.ListTrainings(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, trainings)
}

type trainingExerciseResponse struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Sets       int32     `json:"sets"`
	Reps       string    `json:"reps"`
	Notes      string    `json:"notes"`
}

type trainingBlockResponse struct {
	ID        uuid.UUID                  `json:"id"`
	Name      string                     `json:"name"`
	Notes     string                     `json:"notes"`
	Exercises []trainingExerciseResponse `json:"exercises"`
}

type trainingUnitResponse struct {
	ID       uuid.UUID               `json:"id"`
	Name     string                  `json:"name"`
	Notes    string                  `json:"notes"`
	Weekdays []int16                 `json:"weekdays"`
	Dates    []string                `json:"dates"`
	Blocks   []trainingBlockResponse `json:"blocks"`
}

type trainingResponse struct {
	database.GetTrainingRow
	Units []trainingUnitResponse `json:"units"`
}

// loadTraining reads a training with its units, blocks and exercises in order.
func loadTraining(ctx context.Context, q *database.Queries, id uuid.UUID) (trainingResponse, error) {
	training, err := q.GetTraining(ctx, id)
	if err != nil {
		return trainingResponse{}, err
	}
	units, err := q.ListTrainingUnits(ctx, id)
	if err != nil {
		return trainingResponse{}, err
	}
	blocks, err := q.ListTrainingBlocks(ctx, id)
	if err != nil {
		return trainingResponse{}, err
	}
	exercises, err := q.ListTrainingExercises(ctx, id)
	if err != nil {
		return trainingResponse{}, err
	}

	// The lists come back ordered by position within their parent, so appending keeps order.
	exercisesByBlock := map[uuid.UUID][]trainingExerciseResponse{}
	for _, e := range exercises {
		exercisesByBlock[e.BlockID] = append(exercisesByBlock[e.BlockID], trainingExerciseResponse{
			ExerciseID: e.ExerciseID, Name: e.Name, Type: e.Type, Sets: e.Sets, Reps: e.Reps, Notes: e.Notes,
		})
	}
	blocksByUnit := map[uuid.UUID][]trainingBlockResponse{}
	for _, b := range blocks {
		blocksByUnit[b.UnitID] = append(blocksByUnit[b.UnitID], trainingBlockResponse{
			ID: b.ID, Name: b.Name, Notes: b.Notes, Exercises: orEmpty(exercisesByBlock[b.ID]),
		})
	}
	resp := trainingResponse{GetTrainingRow: training, Units: make([]trainingUnitResponse, 0, len(units))}
	for _, u := range units {
		dates := make([]string, 0, len(u.Dates))
		for _, d := range u.Dates {
			dates = append(dates, d.Time.Format(time.DateOnly))
		}
		resp.Units = append(resp.Units, trainingUnitResponse{
			ID: u.ID, Name: u.Name, Notes: u.Notes, Weekdays: orEmpty(u.Weekdays), Dates: dates,
			Blocks: orEmpty(blocksByUnit[u.ID]),
		})
	}
	return resp, nil
}

func (h *Handler) GetTraining(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	training, err := loadTraining(r.Context(), h.queries, id)
	if err != nil {
		trainingWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, training)
}

// prepareClient checks that the client exists and, when the training becomes active,
// deactivates the client's other trainings (trainingID is uuid.Nil for a new one).
func prepareClient(ctx context.Context, q *database.Queries, clientID, trainingID uuid.UUID, active bool) error {
	if _, err := q.GetUserByID(ctx, clientID); errors.Is(err, pgx.ErrNoRows) {
		return errUnknownClient
	} else if err != nil {
		return err
	}
	if !active {
		return nil
	}
	return q.DeactivateOtherTrainings(ctx, database.DeactivateOtherTrainingsParams{UserID: clientID, ID: trainingID})
}

// orEmpty turns a nil slice into an empty one, so it encodes as [] rather than null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (h *Handler) CreateTraining(w http.ResponseWriter, r *http.Request) {
	req, ok := parseTrainingRequest(w, r)
	if !ok {
		return
	}
	var training database.CreateTrainingRow
	err := h.inTx(r.Context(), func(q *database.Queries) error {
		if err := prepareClient(r.Context(), q, req.ClientID, uuid.Nil, req.Active); err != nil {
			return err
		}
		var err error
		training, err = q.CreateTraining(r.Context(), database.CreateTrainingParams{
			UserID: req.ClientID, Name: req.Name, Description: req.Description, Active: req.Active,
		})
		if err != nil {
			return err
		}
		return saveTrainingUnits(r.Context(), q, training.ID, req.Units)
	})
	if err != nil {
		trainingWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, training)
}

func (h *Handler) UpdateTraining(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	req, ok := parseTrainingRequest(w, r)
	if !ok {
		return
	}
	var training database.UpdateTrainingRow
	err = h.inTx(r.Context(), func(q *database.Queries) error {
		if err := prepareClient(r.Context(), q, req.ClientID, id, req.Active); err != nil {
			return err
		}
		var err error
		training, err = q.UpdateTraining(r.Context(), database.UpdateTrainingParams{
			ID: id, UserID: req.ClientID, Name: req.Name, Description: req.Description, Active: req.Active,
		})
		if err != nil {
			return err
		}
		return saveTrainingUnits(r.Context(), q, id, req.Units)
	})
	if err != nil {
		trainingWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, training)
}

type duplicateTrainingRequest struct {
	ClientID uuid.UUID `json:"client_id"`
}

// DuplicateTraining copies a training, with its units, blocks and exercises, for a client
// (the same or another one). The copy starts inactive.
func (h *Handler) DuplicateTraining(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	var req duplicateTrainingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var training database.CreateTrainingRow
	err = h.inTx(r.Context(), func(q *database.Queries) error {
		source, err := loadTraining(r.Context(), q, id)
		if err != nil {
			return err
		}
		if err := prepareClient(r.Context(), q, req.ClientID, uuid.Nil, false); err != nil {
			return err
		}
		training, err = q.CreateTraining(r.Context(), database.CreateTrainingParams{
			UserID: req.ClientID, Name: source.Name, Description: source.Description,
		})
		if err != nil {
			return err
		}
		return saveTrainingUnits(r.Context(), q, training.ID, copyUnits(source.Units))
	})
	if err != nil {
		trainingWriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, training)
}

// copyUnits turns loaded units into new ones (without ids) for saveTrainingUnits.
func copyUnits(units []trainingUnitResponse) []trainingUnitRequest {
	out := make([]trainingUnitRequest, 0, len(units))
	for _, u := range units {
		unit := trainingUnitRequest{Name: u.Name, Notes: u.Notes, Weekdays: u.Weekdays}
		unit.dates, _ = parseDates(u.Dates) // formatted by loadTraining, so always valid
		for _, b := range u.Blocks {
			block := trainingBlockRequest{Name: b.Name, Notes: b.Notes}
			for _, e := range b.Exercises {
				block.Exercises = append(block.Exercises, trainingExerciseRequest{
					ExerciseID: e.ExerciseID, Sets: e.Sets, Reps: e.Reps, Notes: e.Notes,
				})
			}
			unit.Blocks = append(unit.Blocks, block)
		}
		out = append(out, unit)
	}
	return out
}

// DeleteTraining soft-deletes a training.
func (h *Handler) DeleteTraining(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	n, err := h.queries.DeleteTraining(r.Context(), id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "training not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
