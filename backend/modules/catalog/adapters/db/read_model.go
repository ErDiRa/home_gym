package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/errs"
	"github.com/erdira/homegym/backend/modules/catalog/adapters/db/dbmodels"
	cataloghttp "github.com/erdira/homegym/backend/modules/catalog/api/http"
)

// ReadModel lists what the dashboard and the forms show, mapped straight to the HTTP responses:
// reading is plain data retrieval, no business logic.
type ReadModel struct {
	q *dbmodels.Queries
}

var _ cataloghttp.ReadModel = (*ReadModel)(nil)

func NewReadModel(pool *pgxpool.Pool) *ReadModel {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &ReadModel{q: dbmodels.New(pool)}
}

// Exercises lists the user's exercises with the names of their links; the focuses come as JSON.
func (m *ReadModel) Exercises(ctx context.Context, userID uuid.UUID) ([]cataloghttp.Exercise, error) {
	rows, err := m.q.ListExercises(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]cataloghttp.Exercise, len(rows))
	for i, e := range rows {
		out[i] = cataloghttp.Exercise{
			Id: e.ID, Name: e.Name, Description: e.Description, Type: cataloghttp.ExerciseType(e.Type),
			VideoUrls: e.VideoUrls, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
			Joints: e.Joints, MuscleGroups: e.MuscleGroups,
		}
		if err := json.Unmarshal(e.Focuses, &out[i].Focuses); err != nil {
			return nil, fmt.Errorf("decode focuses of exercise %s: %w", e.ID, err)
		}
	}
	return out, nil
}

// Exercise is one of the user's live exercises with the ids of its links, for the form.
func (m *ReadModel) Exercise(ctx context.Context, userID, id uuid.UUID) (cataloghttp.ExerciseDetail, error) {
	e, err := m.q.GetExercise(ctx, dbmodels.GetExerciseParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return cataloghttp.ExerciseDetail{}, errs.NotFound("exercise not found", err)
	}
	if err != nil {
		return cataloghttp.ExerciseDetail{}, err
	}
	return cataloghttp.ExerciseDetail{
		Id: e.ID, Name: e.Name, Description: e.Description, Type: cataloghttp.ExerciseType(e.Type), VideoUrls: e.VideoUrls,
		JointIds: e.JointIds, MuscleGroupIds: e.MuscleGroupIds, FocusIds: e.FocusIds,
	}, nil
}

// Ratings lists how the user rated their live exercises.
func (m *ReadModel) Ratings(ctx context.Context, userID uuid.UUID) ([]cataloghttp.ExerciseRating, error) {
	rows, err := m.q.ListExerciseRatings(ctx, userID)
	return mapRows(rows, err, func(r dbmodels.ListExerciseRatingsRow) cataloghttp.ExerciseRating {
		return cataloghttp.ExerciseRating{ExerciseId: r.ExerciseID, Rating: r.Rating, Notes: r.Notes, UpdatedAt: r.UpdatedAt}
	})
}

func (m *ReadModel) Joints(ctx context.Context) ([]cataloghttp.Option, error) {
	rows, err := m.q.ListJoints(ctx)
	return mapRows(rows, err, func(j dbmodels.ListJointsRow) cataloghttp.Option { return cataloghttp.Option{Id: j.ID, Name: j.Name} })
}

func (m *ReadModel) MuscleGroups(ctx context.Context) ([]cataloghttp.Option, error) {
	rows, err := m.q.ListMuscleGroups(ctx)
	return mapRows(rows, err, func(g dbmodels.ListMuscleGroupsRow) cataloghttp.Option {
		return cataloghttp.Option{Id: g.ID, Name: g.Name}
	})
}

func (m *ReadModel) Focuses(ctx context.Context) ([]cataloghttp.Focus, error) {
	rows, err := m.q.ListFocuses(ctx)
	return mapRows(rows, err, func(f dbmodels.ListFocusesRow) cataloghttp.Focus {
		return cataloghttp.Focus{Id: f.ID, Name: f.Name, Alias: f.Alias}
	})
}

func (m *ReadModel) BlockKinds(ctx context.Context) ([]cataloghttp.BlockKind, error) {
	rows, err := m.q.ListBlockKinds(ctx)
	return mapRows(rows, err, func(k dbmodels.ListBlockKindsRow) cataloghttp.BlockKind {
		return cataloghttp.BlockKind{
			Code: k.Code, Name: k.Name, Description: k.Description, MinExercises: k.MinExercises, MaxExercises: k.MaxExercises,
			SharedSets: k.SharedSets, SharedReps: k.SharedReps, SharedLoad: k.SharedLoad, Scored: k.Scored,
		}
	})
}

func (m *ReadModel) RepUnits(ctx context.Context) ([]cataloghttp.Unit, error) {
	rows, err := m.q.ListRepUnits(ctx)
	return mapRows(rows, err, func(u dbmodels.ListRepUnitsRow) cataloghttp.Unit {
		return cataloghttp.Unit{Id: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue}
	})
}

func (m *ReadModel) LoadUnits(ctx context.Context) ([]cataloghttp.LoadUnit, error) {
	rows, err := m.q.ListLoadUnits(ctx)
	return mapRows(rows, err, func(u dbmodels.ListLoadUnitsRow) cataloghttp.LoadUnit {
		return cataloghttp.LoadUnit{
			Id: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue, KgPerUnit: u.KgPerUnit, PercentOfMax: u.PercentOfMax,
		}
	})
}

// mapRows maps the rows of a query with f, or passes its error on. The result is never nil, so
// it encodes as [].
func mapRows[T, U any](rows []T, err error, f func(T) U) ([]U, error) {
	if err != nil {
		return nil, err
	}
	out := make([]U, len(rows))
	for i, r := range rows {
		out[i] = f(r)
	}
	return out, nil
}
