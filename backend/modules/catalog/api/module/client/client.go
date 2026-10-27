// Package client is the catalog module's contract: what other modules may read of the catalog.
package client

import (
	"context"

	"github.com/google/uuid"
)

type Catalog interface {
	// Exercises returns the user's exercises among ids, keyed by id. Deleted ones are included
	// (marked), as plans and logs still show them; an id that isn't the user's is missing.
	Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]Exercise, error)
	// BlockKinds returns the kinds of training blocks with their rules, in display order.
	BlockKinds(ctx context.Context) ([]BlockKind, error)
	// Units returns every unit of reps and load, deleted ones too (marked).
	Units(ctx context.Context) (Units, error)
	// ExerciseProfiles returns the user's live exercises among ids with their joints, muscle
	// groups and the user's rating, keyed by id.
	ExerciseProfiles(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]ExerciseProfile, error)
}

// ExerciseProfile is an exercise as stats show it.
type ExerciseProfile struct {
	ID           uuid.UUID
	Name         string
	Joints       []string
	MuscleGroups []string
	Rating       *int32 // 1-5; nil if unrated
}

type Exercise struct {
	ID      uuid.UUID
	Name    string
	Type    string // mobility or strength
	Deleted bool
}

// BlockKind is a kind of training block (single, superset, complex, wod) with its rules.
type BlockKind struct {
	Code         string
	Name         string
	MinExercises int32
	MaxExercises *int32 // nil = no limit
	SharedSets   bool   // the sets are set once on the block, for all its exercises
	SharedReps   bool
	SharedLoad   bool
	Scored       bool // done for one score (a WOD); its shared sets are rounds and optional
}

// Unit is a unit of planned reps or load. MinValue/MaxValue bound the values planned in it.
// Only load units have KgPerUnit (weight units: kg 1, lb 0.45359237) and PercentOfMax (% 1RM).
type Unit struct {
	ID           uuid.UUID
	Name         string
	MinValue     *float64
	MaxValue     *float64
	KgPerUnit    *float64
	PercentOfMax bool
	Deleted      bool
}

type Units struct {
	Rep  []Unit
	Load []Unit
}
