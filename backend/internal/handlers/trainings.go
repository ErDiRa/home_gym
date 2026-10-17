package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
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
// and/or dates) hold blocks, and blocks hold one or more exercises. A block has a kind
// (single, superset, complex, see block_kinds) that decides how many exercises it takes and
// whether reps and load are set once on the block or per exercise; its sets apply to all of
// its exercises. The same shape is sent to and returned from the API.

// amount is the planned reps and load of a block or exercise. A value is nil when not set;
// a max turns it into a range (e.g. 8-10 reps). Units are rep_units / load_units ids.
type amount struct {
	RepsValue *float64   `json:"reps_value"`
	RepsMax   *float64   `json:"reps_max"`
	RepsUnit  *uuid.UUID `json:"reps_unit"`
	PerSide   bool       `json:"per_side"`
	LoadValue *float64   `json:"load_value"`
	LoadMax   *float64   `json:"load_max"`
	LoadUnit  *uuid.UUID `json:"load_unit"`
}

func (a *amount) clearReps() { a.RepsValue, a.RepsMax, a.RepsUnit, a.PerSide = nil, nil, nil, false }
func (a *amount) clearLoad() { a.LoadValue, a.LoadMax, a.LoadUnit = nil, nil, nil }

type trainingExerciseRequest struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	amount
	Notes string `json:"notes"`
}

type trainingBlockRequest struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Sets int32  `json:"sets"`
	amount
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

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000

// parseTrainingRequest decodes, trims and validates a training body against the lookups,
// writing the error response itself on failure.
func parseTrainingRequest(w http.ResponseWriter, r *http.Request, lk trainingLookups) (trainingRequest, bool) {
	var req trainingRequest
	if !decodeJSON(w, r, &req) {
		return req, false
	}
	fail := func(msg string) (trainingRequest, bool) {
		writeError(w, r, http.StatusBadRequest, msg)
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
			if msg := checkBlock(&u.Blocks[j], lk); msg != "" {
				return fail(fmt.Sprintf("%s, block %s: %s", u.Name, blockLetter(j), msg))
			}
		}
	}
	return req, true
}

// checkBlock trims and validates a block against its kind, clearing reps and load on the side
// (block or exercises) the kind doesn't use. It returns a message for the first problem.
func checkBlock(b *trainingBlockRequest, lk trainingLookups) string {
	b.Name, b.Notes = strings.TrimSpace(b.Name), strings.TrimSpace(b.Notes)
	kind, ok := lk.kinds[b.Kind]
	if !ok {
		return "unknown block kind"
	}
	if tooLong(b.Name, 100) {
		return "the name can have at most 100 characters"
	}
	if b.Sets < 1 || b.Sets > 100 {
		return "sets must be 1-100"
	}
	n := int32(len(b.Exercises))
	if n < kind.MinExercises || (kind.MaxExercises.Valid && n > kind.MaxExercises.Int32) {
		return exerciseCountRule(kind)
	}

	if kind.SharedReps {
		if msg := checkReps(&b.amount, lk); msg != "" {
			return msg
		}
	} else {
		b.clearReps()
	}
	if kind.SharedLoad {
		if msg := checkLoad(&b.amount, lk); msg != "" {
			return msg
		}
	} else {
		b.clearLoad()
	}
	for k := range b.Exercises {
		e := &b.Exercises[k]
		e.Notes = strings.TrimSpace(e.Notes)
		if e.ExerciseID == uuid.Nil {
			return "every exercise needs an exercise_id"
		}
		if kind.SharedReps {
			e.clearReps()
		} else if msg := checkReps(&e.amount, lk); msg != "" {
			return fmt.Sprintf("exercise %d: %s", k+1, msg)
		}
		if kind.SharedLoad {
			e.clearLoad()
		} else if msg := checkLoad(&e.amount, lk); msg != "" {
			return fmt.Sprintf("exercise %d: %s", k+1, msg)
		}
	}
	return ""
}

// exerciseCountRule describes how many exercises a kind of block takes.
func exerciseCountRule(kind database.ListBlockKindsRow) string {
	switch {
	case !kind.MaxExercises.Valid:
		return fmt.Sprintf("a %s block needs at least %d exercises", kind.Name, kind.MinExercises)
	case kind.MaxExercises.Int32 == kind.MinExercises:
		return fmt.Sprintf("a %s block takes exactly %d %s", kind.Name, kind.MinExercises, plural(kind.MinExercises, "exercise"))
	default:
		return fmt.Sprintf("a %s block takes %d to %d exercises", kind.Name, kind.MinExercises, kind.MaxExercises.Int32)
	}
}

func plural(n int32, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// checkReps validates the required reps of an amount.
func checkReps(a *amount, lk trainingLookups) string {
	if a.RepsValue == nil {
		return "reps are required"
	}
	return checkRange("reps", a.RepsValue, a.RepsMax, a.RepsUnit, lk.repUnits)
}

// checkLoad validates the optional load of an amount.
func checkLoad(a *amount, lk trainingLookups) string {
	if a.LoadValue == nil && a.LoadMax == nil && a.LoadUnit == nil {
		return ""
	}
	return checkRange("load", a.LoadValue, a.LoadMax, a.LoadUnit, lk.loadUnits)
}

// checkRange validates a value with an optional max in one of units.
func checkRange(what string, value, max *float64, unitID *uuid.UUID, units map[uuid.UUID]unit) string {
	if value == nil || unitID == nil {
		return what + " needs a value and a unit"
	}
	u, ok := units[*unitID]
	if !ok {
		return "unknown " + what + " unit"
	}
	if max != nil && *max <= *value {
		return what + ": the upper end of a range must be higher than the value"
	}
	for _, v := range []*float64{value, max} {
		if v == nil {
			continue
		}
		if *v < 0 || *v >= maxAmount || (u.min != nil && *v < *u.min) || (u.max != nil && *v > *u.max) {
			return fmt.Sprintf("%s in %s must be %s", what, u.name, boundsText(u))
		}
	}
	return ""
}

// boundsText describes the values a unit allows, e.g. "between 1 and 10".
func boundsText(u unit) string {
	lo, hi := 0.0, float64(maxAmount-1)
	if u.min != nil {
		lo = *u.min
	}
	if u.max != nil {
		hi = *u.max
	}
	return fmt.Sprintf("between %g and %g", lo, hi)
}

// blockLetter labels a block by its position in the unit: A, B, ... Z, then 27, 28...
func blockLetter(i int) string {
	if i < 26 {
		return string(rune('A' + i))
	}
	return strconv.Itoa(i + 1)
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
				UnitID: unitID, Position: int32(j + 1), Kind: b.Kind, Name: b.Name, Sets: b.Sets,
				RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
				LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit, Notes: b.Notes,
			})
			if err != nil {
				return err
			}
			for k, e := range b.Exercises {
				err := q.InsertTrainingExercise(ctx, database.InsertTrainingExerciseParams{
					BlockID: blockID, ExerciseID: e.ExerciseID, Position: int32(k + 1),
					RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
					LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, Notes: e.Notes,
				})
				if err != nil {
					return err
				}
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
		clientError(w, r, http.StatusNotFound, "training not found", err)
	case errors.Is(err, errUnknownUnit):
		clientError(w, r, http.StatusBadRequest, "unknown unit", err)
	case errors.Is(err, errUnknownClient):
		clientError(w, r, http.StatusBadRequest, "unknown client", err)
	case errors.As(err, &pgErr) && pgErr.Code == "23503": // foreign_key_violation
		clientError(w, r, http.StatusBadRequest, "unknown exercise", err)
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

// amountResponse is an amount with the names of its units, for display.
type amountResponse struct {
	amount
	RepsUnitName string `json:"reps_unit_name"`
	LoadUnitName string `json:"load_unit_name"`
}

type trainingExerciseResponse struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	amountResponse
	Notes string `json:"notes"`
}

type trainingBlockResponse struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"`
	Name string    `json:"name"`
	Sets int32     `json:"sets"`
	amountResponse
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
			ExerciseID: e.ExerciseID, Name: e.Name, Type: e.Type, Notes: e.Notes,
			amountResponse: amountResponse{
				amount: amount{
					RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
					LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit,
				},
				RepsUnitName: e.RepsUnitName, LoadUnitName: e.LoadUnitName,
			},
		})
	}
	blocksByUnit := map[uuid.UUID][]trainingBlockResponse{}
	for _, b := range blocks {
		blocksByUnit[b.UnitID] = append(blocksByUnit[b.UnitID], trainingBlockResponse{
			ID: b.ID, Kind: b.Kind, Name: b.Name, Sets: b.Sets, Notes: b.Notes,
			amountResponse: amountResponse{
				amount: amount{
					RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
					LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit,
				},
				RepsUnitName: b.RepsUnitName, LoadUnitName: b.LoadUnitName,
			},
			Exercises: orEmpty(exercisesByBlock[b.ID]),
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
		clientError(w, r, http.StatusBadRequest, "invalid training id", err)
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
	lk, err := loadTrainingLookups(r.Context(), h.queries)
	if err != nil {
		serverError(w, r, err)
		return
	}
	req, ok := parseTrainingRequest(w, r, lk)
	if !ok {
		return
	}
	var training database.CreateTrainingRow
	err = h.inTx(r.Context(), func(q *database.Queries) error {
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
		clientError(w, r, http.StatusBadRequest, "invalid training id", err)
		return
	}
	lk, err := loadTrainingLookups(r.Context(), h.queries)
	if err != nil {
		serverError(w, r, err)
		return
	}
	req, ok := parseTrainingRequest(w, r, lk)
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
		clientError(w, r, http.StatusBadRequest, "invalid training id", err)
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
			block := trainingBlockRequest{Kind: b.Kind, Name: b.Name, Sets: b.Sets, amount: b.amount, Notes: b.Notes}
			for _, e := range b.Exercises {
				block.Exercises = append(block.Exercises, trainingExerciseRequest{
					ExerciseID: e.ExerciseID, amount: e.amount, Notes: e.Notes,
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
		clientError(w, r, http.StatusBadRequest, "invalid training id", err)
		return
	}
	n, err := h.queries.DeleteTraining(r.Context(), id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if n == 0 {
		writeError(w, r, http.StatusNotFound, "training not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
