package db

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	"github.com/erdira/homegym/backend/modules/tracking/adapters/db/dbmodels"
	trackinghttp "github.com/erdira/homegym/backend/modules/tracking/api/http"
)

// Catalog is what the read model reads of the catalog module (its contract): the names of
// exercises and units, and the exercises' profiles for the stats.
type Catalog interface {
	Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error)
	Units(ctx context.Context) (catalog.Units, error)
	ExerciseProfiles(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.ExerciseProfile, error)
}

// ReadModel lists what the tracker and the dashboard show, mapped straight to the HTTP responses:
// reading is plain data retrieval, no business logic. The names of exercises and units are the
// catalog's, read through its contract.
type ReadModel struct {
	q       *dbmodels.Queries
	catalog Catalog
}

var _ trackinghttp.ReadModel = (*ReadModel)(nil)

func NewReadModel(pool *pgxpool.Pool, catalog Catalog) *ReadModel {
	if pool == nil {
		panic("pool cannot be nil")
	}
	if catalog == nil {
		panic("catalog cannot be nil")
	}
	return &ReadModel{q: dbmodels.New(pool), catalog: catalog}
}

// Workouts lists the user's workouts finished in [from, to), newest first.
func (m *ReadModel) Workouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]trackinghttp.WorkoutSummary, error) {
	rows, err := m.q.ListWorkouts(ctx, dbmodels.ListWorkoutsParams{UserID: userID, FromTime: &from, ToTime: &to})
	if err != nil {
		return nil, err
	}
	return mapSlice(rows, func(w dbmodels.ListWorkoutsRow) trackinghttp.WorkoutSummary {
		return trackinghttp.WorkoutSummaryToAPI(workoutSummary(w))
	}), nil
}

// Workout is one of the user's live workouts with its sets (with the names of their exercise and
// units) and WOD scores, in the order of the session.
func (m *ReadModel) Workout(ctx context.Context, userID, id uuid.UUID) (trackinghttp.Workout, error) {
	w, err := m.q.GetWorkout(ctx, dbmodels.GetWorkoutParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return trackinghttp.Workout{}, errs.NotFound("workout not found", err)
	}
	if err != nil {
		return trackinghttp.Workout{}, err
	}
	sets, err := m.q.ListWorkoutSets(ctx, id)
	if err != nil {
		return trackinghttp.Workout{}, err
	}
	results, err := m.q.ListWorkoutResults(ctx, id)
	if err != nil {
		return trackinghttp.Workout{}, err
	}
	exerciseIDs := make([]uuid.UUID, len(sets))
	for i, s := range sets {
		exerciseIDs[i] = s.ExerciseID
	}
	exercises, err := m.catalog.Exercises(ctx, userID, exerciseIDs)
	if err != nil {
		return trackinghttp.Workout{}, err
	}
	repUnits, loadUnits, err := m.unitNames(ctx)
	if err != nil {
		return trackinghttp.Workout{}, err
	}
	return trackinghttp.Workout{
		Id: w.ID, TrainingId: w.TrainingID, WeekSessionId: w.WeekSessionID, SessionId: w.SessionID, SessionName: w.SessionName,
		StartedAt: w.StartedAt, CompletedAt: w.CompletedAt, Notes: w.Notes,
		Sets: mapSlice(sets, func(s dbmodels.TrackingWorkoutSet) trackinghttp.WorkoutSet {
			loadUnitName := ""
			if s.LoadUnit != nil {
				loadUnitName = loadUnits[*s.LoadUnit]
			}
			return trackinghttp.WorkoutSet{
				Id: s.ID, ExerciseId: s.ExerciseID, ExerciseName: exercises[s.ExerciseID].Name,
				BlockPosition: s.BlockPosition, ExercisePosition: s.ExercisePosition, SetNumber: s.SetNumber,
				RepsValue: s.RepsValue, RepsUnit: s.RepsUnit, RepsUnitName: repUnits[s.RepsUnit], PerSide: s.PerSide,
				LoadValue: s.LoadValue, LoadUnit: s.LoadUnit, LoadUnitName: loadUnitName, Bodyweight: s.Bodyweight,
				CompletedAt: s.CompletedAt,
			}
		}),
		Results: mapSlice(results, func(r dbmodels.TrackingWorkoutResult) trackinghttp.WorkoutResult {
			return trackinghttp.WorkoutResult{
				Id: r.ID, WorkoutId: r.WorkoutID, BlockPosition: r.BlockPosition, ScoreType: trackinghttp.ScoreType(r.ScoreType),
				TimeSeconds: r.TimeSeconds, Rounds: r.Rounds, Reps: r.Reps, Calories: r.Calories, Rx: r.Rx, Notes: r.Notes,
				CompletedAt: r.CompletedAt,
			}
		}),
	}, nil
}

// Maxes lists the 1RM history of the user's live exercises, newest first per exercise.
func (m *ReadModel) Maxes(ctx context.Context, userID uuid.UUID) ([]trackinghttp.ExerciseMax, error) {
	rows, err := m.q.ListExerciseMaxes(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ExerciseID
	}
	exercises, err := m.catalog.Exercises(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	_, loadUnits, err := m.unitNames(ctx)
	if err != nil {
		return nil, err
	}
	out := []trackinghttp.ExerciseMax{}
	for _, r := range rows {
		if e, ok := exercises[r.ExerciseID]; ok && !e.Deleted {
			out = append(out, maxToAPI(dbmodels.GetExerciseMaxRow(r), loadUnits))
		}
	}
	return out, nil
}

// Max is one entry of the user's 1RM history.
func (m *ReadModel) Max(ctx context.Context, userID, id uuid.UUID) (trackinghttp.ExerciseMax, error) {
	r, err := m.q.GetExerciseMax(ctx, dbmodels.GetExerciseMaxParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return trackinghttp.ExerciseMax{}, errs.NotFound("1RM not found", err)
	}
	if err != nil {
		return trackinghttp.ExerciseMax{}, err
	}
	_, loadUnits, err := m.unitNames(ctx)
	if err != nil {
		return trackinghttp.ExerciseMax{}, err
	}
	return maxToAPI(r, loadUnits), nil
}

func maxToAPI(r dbmodels.GetExerciseMaxRow, loadUnits map[uuid.UUID]string) trackinghttp.ExerciseMax {
	return trackinghttp.ExerciseMax{
		Id: r.ID, ExerciseId: r.ExerciseID, LoadValue: r.LoadValue, LoadUnit: r.LoadUnit, LoadUnitName: loadUnits[r.LoadUnit],
		AchievedOn: openapi_types.Date{Time: r.AchievedOn}, WorkoutSetId: r.WorkoutSetID, Notes: r.Notes, CreatedAt: r.CreatedAt,
	}
}

// ExerciseStats lists the user's live exercises done in finished workouts, most often first (then
// by name), with in how many workouts and the user's rating.
func (m *ReadModel) ExerciseStats(ctx context.Context, userID uuid.UUID) ([]trackinghttp.ExerciseStat, error) {
	counts, err := m.q.CountExercisesDone(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(counts))
	for i, c := range counts {
		ids[i] = c.ExerciseID
	}
	profiles, err := m.catalog.ExerciseProfiles(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := []trackinghttp.ExerciseStat{}
	for _, c := range counts {
		if p, ok := profiles[c.ExerciseID]; ok {
			out = append(out, trackinghttp.ExerciseStat{
				ExerciseId: c.ExerciseID, Name: p.Name, Joints: p.Joints, MuscleGroups: p.MuscleGroups,
				TimesPerformed: c.TimesPerformed, Rating: p.Rating,
			})
		}
	}
	slices.SortFunc(out, func(a, b trackinghttp.ExerciseStat) int {
		if a.TimesPerformed != b.TimesPerformed {
			return int(b.TimesPerformed - a.TimesPerformed)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out, nil
}

// LastLoads lists the load the user last logged of each of their live exercises.
func (m *ReadModel) LastLoads(ctx context.Context, userID uuid.UUID) ([]trackinghttp.LastLoad, error) {
	rows, err := m.q.ListLastLoads(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ExerciseID
	}
	exercises, err := m.catalog.Exercises(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	_, loadUnits, err := m.unitNames(ctx)
	if err != nil {
		return nil, err
	}
	out := []trackinghttp.LastLoad{}
	for _, r := range rows {
		// The query only takes sets with a load, which always have a unit.
		if e, ok := exercises[r.ExerciseID]; !ok || e.Deleted || r.LoadValue == nil || r.LoadUnit == nil || r.CompletedAt == nil {
			continue
		}
		out = append(out, trackinghttp.LastLoad{
			ExerciseId: r.ExerciseID, LoadValue: *r.LoadValue, LoadUnit: *r.LoadUnit, LoadUnitName: loadUnits[*r.LoadUnit],
			DoneAt: *r.CompletedAt,
		})
	}
	return out, nil
}

// unitNames returns the names of the rep and load units by id.
func (m *ReadModel) unitNames(ctx context.Context) (rep, load map[uuid.UUID]string, err error) {
	units, err := m.catalog.Units(ctx)
	if err != nil {
		return nil, nil, err
	}
	rep, load = map[uuid.UUID]string{}, map[uuid.UUID]string{}
	for _, u := range units.Rep {
		rep[u.ID] = u.Name
	}
	for _, u := range units.Load {
		load[u.ID] = u.Name
	}
	return rep, load, nil
}

// mapSlice maps every element with f; the result is never nil, so it encodes as [].
func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}
