package maxes

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

// MaxInput is a 1RM: the load in a weight unit, the day it was lifted and, when it was saved
// from a logged single, that set.
type MaxInput struct {
	LoadValue    float64    `json:"load_value"`
	LoadUnit     uuid.UUID  `json:"load_unit"`
	AchievedOn   string     `json:"achieved_on"` // YYYY-MM-DD
	WorkoutSetID *uuid.UUID `json:"workout_set_id"`
	Notes        string     `json:"notes"`
}

// maxAmount is the exclusive upper bound of a load (NUMERIC(7, 2) column).
const maxAmount = 100000

// Add adds a 1RM to the history of one of the user's exercises; being the newest, it becomes the
// current max. Earlier entries stay, so the history shows the progress.
func (s *Service) Add(ctx context.Context, userID, exerciseID uuid.UUID, in MaxInput) (repos.CreateExerciseMaxRow, error) {
	day, err := s.check(ctx, userID, exerciseID, &in)
	if err != nil {
		return repos.CreateExerciseMaxRow{}, err
	}
	created, err := s.q.CreateExerciseMax(ctx, repos.CreateExerciseMaxParams{
		UserID: userID, ExerciseID: exerciseID,
		LoadValue: in.LoadValue, LoadUnit: in.LoadUnit, AchievedOn: day,
		WorkoutSetID: in.WorkoutSetID, Notes: in.Notes,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return repos.CreateExerciseMaxRow{}, service.NotFound("exercise not found", err)
	}
	return created, err
}

// check validates a 1RM, trimming its notes in place, and returns its day.
func (s *Service) check(ctx context.Context, userID, exerciseID uuid.UUID, in *MaxInput) (pgtype.Date, error) {
	if in.LoadValue <= 0 || in.LoadValue >= maxAmount {
		return pgtype.Date{}, service.Invalid("load must be between 0 and 99999")
	}
	day, err := time.Parse(time.DateOnly, in.AchievedOn)
	if err != nil {
		return pgtype.Date{}, service.Invalid("achieved_on must be a date (YYYY-MM-DD)")
	}
	// The client sends its local day, which may be a day ahead of the server's.
	if day.Year() < 1900 || day.After(time.Now().AddDate(0, 0, 1)) {
		return pgtype.Date{}, service.Invalid("achieved_on must not be in the future")
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if utf8.RuneCountInString(in.Notes) > 1000 {
		return pgtype.Date{}, service.Invalid("notes can have at most 1000 characters")
	}

	unit, err := s.q.GetLoadUnit(ctx, in.LoadUnit)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.Date{}, service.InvalidCause("unknown load unit", err)
	}
	if err != nil {
		return pgtype.Date{}, err
	}
	if unit.KgPerUnit == nil {
		return pgtype.Date{}, service.Invalid("a 1RM needs a weight unit such as kg or lb")
	}

	if in.WorkoutSetID != nil {
		owns, err := s.q.UserOwnsSet(ctx, repos.UserOwnsSetParams{UserID: userID, ID: *in.WorkoutSetID, ExerciseID: exerciseID})
		if err != nil {
			return pgtype.Date{}, err
		}
		if !owns {
			return pgtype.Date{}, service.Invalid("unknown set")
		}
	}
	return pgtype.Date{Time: day, Valid: true}, nil
}
