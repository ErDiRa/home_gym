// Package app holds the tracking module's business logic: the tracker's week, workouts with their
// sets and WOD scores, and the 1RM history.
//
// A training (the planning module's) plans the sessions of each of its weeks instead of fixed
// days. A workout is one session done: the user starts it from a planned slot of the week whenever
// there is time, logs every set as they go (and one score per WOD block) and finishes it. Several
// trainings can be active side by side (e.g. strength, warm-ups and mobility); the week shows one
// of them at a time: its planned sessions of the week next to the finished workouts, which mark
// their slots done. Exercises and units are the catalog's; plans the planning module's, both read
// through their contracts.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
)

// Planning is what tracking reads of the planning module (its contract).
type Planning interface {
	ActiveTrainings(ctx context.Context, userID uuid.UUID) ([]planning.ActiveTraining, error)
	WeekPlan(ctx context.Context, userID, trainingID uuid.UUID, week int32) ([]planning.PlannedSlot, error)
	Slot(ctx context.Context, userID, slotID uuid.UUID) (planning.Slot, error)
	Cycle(ctx context.Context, userID, cycleID uuid.UUID) (planning.Cycle, error)
}

// Catalog is what tracking reads of the catalog module (its contract).
type Catalog interface {
	Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error)
	Units(ctx context.Context) (catalog.Units, error)
}

var (
	// ErrNotFound is a repository's answer for a missing workout, set, score or 1RM (or another
	// user's).
	ErrNotFound = errors.New("not found")
	// ErrSlotInProgress is a repository's answer when a slot already has a workout in progress.
	ErrSlotInProgress = errors.New("slot in progress")
)

// WorkoutRepository saves workouts with their sets and scores.
type WorkoutRepository interface {
	// CreateWorkout starts a workout of a slot, or returns ErrSlotInProgress.
	CreateWorkout(ctx context.Context, userID uuid.UUID, slot planning.Slot) (uuid.UUID, error)
	// SlotWorkoutInProgress returns the slot's workout in progress, or ErrNotFound.
	SlotWorkoutInProgress(ctx context.Context, userID, slotID uuid.UUID) (uuid.UUID, error)
	// FinishWorkout sets the completion time (completedAt, or now if nil and not finished yet)
	// and the notes, or returns ErrNotFound.
	FinishWorkout(ctx context.Context, userID, id uuid.UUID, notes string, completedAt *time.Time) error
	DeleteWorkout(ctx context.Context, userID, id uuid.UUID) error
	// FinishedWorkouts returns the user's workouts finished in [from, to), newest first.
	FinishedWorkouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]WorkoutSummary, error)
	UserOwnsWorkout(ctx context.Context, userID, id uuid.UUID) (bool, error)
	LogSet(ctx context.Context, workoutID uuid.UUID, in SetInput) (LoggedSet, error)
	// DeleteSet removes a set of the workout, or returns ErrNotFound.
	DeleteSet(ctx context.Context, workoutID, setID uuid.UUID) error
	LogResult(ctx context.Context, workoutID uuid.UUID, in ResultInput) (Result, error)
	// DeleteResult removes a score of the workout, or returns ErrNotFound.
	DeleteResult(ctx context.Context, workoutID, resultID uuid.UUID) error
}

// MaxRepository saves the 1RM history.
type MaxRepository interface {
	CreateMax(ctx context.Context, userID, exerciseID uuid.UUID, in MaxInput, achievedOn time.Time) (uuid.UUID, error)
	// UserOwnsSet reports whether the set was logged for the exercise in one of the user's live
	// workouts.
	UserOwnsSet(ctx context.Context, userID, setID, exerciseID uuid.UUID) (bool, error)
	// DeleteMax soft-deletes one of the user's 1RMs, or returns ErrNotFound.
	DeleteMax(ctx context.Context, userID, id uuid.UUID) error
}

// Service implements the tracking usecases.
type Service struct {
	workouts WorkoutRepository
	maxes    MaxRepository
	planning Planning
	catalog  Catalog
	now      func() time.Time
}

func NewService(workouts WorkoutRepository, maxes MaxRepository, planning Planning, catalog Catalog) *Service {
	if workouts == nil {
		panic("workouts cannot be nil")
	}
	if maxes == nil {
		panic("maxes cannot be nil")
	}
	if planning == nil {
		panic("planning cannot be nil")
	}
	if catalog == nil {
		panic("catalog cannot be nil")
	}
	return &Service{workouts: workouts, maxes: maxes, planning: planning, catalog: catalog, now: time.Now}
}

// WorkoutSummary is a finished workout as listed.
type WorkoutSummary struct {
	ID            uuid.UUID
	TrainingID    *uuid.UUID // nil once the training is deleted
	WeekSessionID *uuid.UUID // the slot; nil once it is removed
	SessionID     *uuid.UUID // nil once the session is removed
	SessionName   string
	StartedAt     time.Time
	CompletedAt   time.Time
	Notes         string
	SetCount      int64
}

// LoggedSet is a set as stored.
type LoggedSet struct {
	ID               uuid.UUID
	WorkoutID        uuid.UUID
	ExerciseID       uuid.UUID
	BlockPosition    int32
	ExercisePosition int32
	SetNumber        int32
	RepsValue        *float64
	RepsUnit         uuid.UUID
	PerSide          bool
	LoadValue        *float64
	LoadUnit         *uuid.UUID
	CompletedAt      time.Time
}

// Result is the score of a WOD block as stored.
type Result struct {
	ID            uuid.UUID
	WorkoutID     uuid.UUID
	BlockPosition int32
	ScoreType     string
	TimeSeconds   *int32
	Rounds        *int32
	Reps          *int32
	Calories      *int32
	Rx            bool
	Notes         string
	CompletedAt   time.Time
}

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000
