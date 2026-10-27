// Package app holds the catalog module's business logic: validating and saving a user's exercises
// and ratings and the shared lists (focuses, units). Reading them is the read model's
// (adapters/db), as it is plain data retrieval.
package app

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Exercise is an exercise as saved.
type Exercise struct {
	ID          uuid.UUID
	Name        string
	Description string
	Type        string // mobility or strength
	VideoURLs   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Rating is a user's rating (1-5) of one of their exercises.
type Rating struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ExerciseID uuid.UUID
	Rating     int32
	Notes      string
	UpdatedAt  time.Time
}

// Focus is a training focus exercises belong to, shared by all users.
type Focus struct {
	ID    uuid.UUID
	Name  string
	Alias string
}

// Unit is a unit of planned reps or load, shared by all users. Only load units have KgPerUnit
// (weight units) and PercentOfMax (% 1RM); units users add have neither.
type Unit struct {
	ID           uuid.UUID
	Name         string
	MinValue     *float64
	MaxValue     *float64
	KgPerUnit    *float64
	PercentOfMax bool
}

var (
	// ErrNotFound is a repository's answer for a missing exercise (or another user's).
	ErrNotFound = errors.New("not found")
	// ErrUnknownReference is a repository's answer when a linked id doesn't exist.
	ErrUnknownReference = errors.New("unknown reference")
	// ErrDuplicate is a repository's answer when a name is taken.
	ErrDuplicate = errors.New("duplicate")
)

// ExerciseRepository saves a user's exercises with their links and videos.
type ExerciseRepository interface {
	CreateExercise(ctx context.Context, userID uuid.UUID, in ExerciseInput) (Exercise, error)
	// UpdateExercise replaces the user's live exercise id, or returns ErrNotFound.
	UpdateExercise(ctx context.Context, userID, id uuid.UUID, in ExerciseInput) (Exercise, error)
}

// RatingRepository saves ratings.
type RatingRepository interface {
	// RateExercise stores the rating of the user's live exercise, replacing an earlier one, or
	// returns ErrNotFound.
	RateExercise(ctx context.Context, userID, exerciseID uuid.UUID, in RatingInput) (Rating, error)
}

// LookupRepository adds to the shared lists; a taken name is ErrDuplicate.
type LookupRepository interface {
	CreateFocus(ctx context.Context, in FocusInput) (Focus, error)
	CreateRepUnit(ctx context.Context, in UnitInput) (Unit, error)
	CreateLoadUnit(ctx context.Context, in UnitInput) (Unit, error)
}

// Service implements the catalog usecases.
type Service struct {
	exercises ExerciseRepository
	ratings   RatingRepository
	lookups   LookupRepository
}

func NewService(exercises ExerciseRepository, ratings RatingRepository, lookups LookupRepository) *Service {
	if exercises == nil {
		panic("exercises cannot be nil")
	}
	if ratings == nil {
		panic("ratings cannot be nil")
	}
	if lookups == nil {
		panic("lookups cannot be nil")
	}
	return &Service{exercises: exercises, ratings: ratings, lookups: lookups}
}

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }
