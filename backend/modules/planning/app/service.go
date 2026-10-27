// Package app holds the planning module's business logic: validating and saving a user's
// trainings, library sessions and cycles. Reading them is the read model's (adapters/db).
//
// A training belongs to one user and runs over whole weeks from a Monday. It has its own sessions
// (e.g. "Session 1"), which hold blocks, and blocks hold one or more exercises, plus a schedule:
// the sessions planned for each week, up to 7, repeats allowed. A block has a kind (single,
// superset, complex, wod: the catalog's block kinds) that decides how many exercises it takes,
// whether sets, reps and load are set once on the block or per exercise, and whether the block is
// done for one score (a WOD). Exercises and units are the catalog's, checked through its contract.
//
// A training may belong to one of the user's cycles: a planned period of whole weeks with goals.
// The user's library holds session templates of the same shape; adding one to a training copies
// it (the client sends the copy), so the two are independent afterwards. Every usecase acts on the
// given user's data only; another user's looks missing.
package app

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
)

// Catalog is what planning reads of the catalog module (its contract).
type Catalog interface {
	Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error)
	BlockKinds(ctx context.Context) ([]catalog.BlockKind, error)
	Units(ctx context.Context) (catalog.Units, error)
}

var (
	// ErrNotFound is a repository's answer for a missing training, session or cycle (or another
	// user's).
	ErrNotFound = errors.New("not found")
	// ErrUnknownCycle is a repository's answer when a training's cycle isn't one of the user's
	// live cycles.
	ErrUnknownCycle = errors.New("unknown cycle")
	// ErrUnknownReference is a repository's answer when a referenced row (e.g. an exercise deleted
	// meanwhile) doesn't exist.
	ErrUnknownReference = errors.New("unknown reference")
)

// TrainingRepository saves trainings with their whole tree, each in one transaction.
type TrainingRepository interface {
	// CreateTraining saves a new training of the user; its cycle must be one of the user's live
	// cycles (else ErrUnknownCycle).
	CreateTraining(ctx context.Context, userID uuid.UUID, in Input) (TrainingHeader, error)
	// UpdateTraining loads the user's training id (ErrNotFound if missing) and replaces it with
	// what update returns, in one transaction; an error from update rolls it back.
	UpdateTraining(ctx context.Context, userID, id uuid.UUID, update func(ctx context.Context, current Training) (Input, error)) (TrainingHeader, error)
	// Training loads one of the user's trainings, or ErrNotFound.
	Training(ctx context.Context, userID, id uuid.UUID) (Training, error)
	DeleteTraining(ctx context.Context, userID, id uuid.UUID) error
}

// LibraryRepository saves the user's session templates.
type LibraryRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID, in SessionInput) (uuid.UUID, error)
	// UpdateSession replaces a template's fields and blocks, or returns ErrNotFound.
	UpdateSession(ctx context.Context, userID, id uuid.UUID, in SessionInput) error
	DeleteSession(ctx context.Context, userID, id uuid.UUID) error
}

// CycleRepository saves the user's cycles with their goals.
type CycleRepository interface {
	CreateCycle(ctx context.Context, userID uuid.UUID, in CycleInput) (uuid.UUID, error)
	// UpdateCycle replaces a cycle's fields and goals, or returns ErrNotFound.
	UpdateCycle(ctx context.Context, userID, id uuid.UUID, in CycleInput) error
	// DeleteCycle soft-deletes a cycle and its trainings (withTrainings) or takes them out of it.
	DeleteCycle(ctx context.Context, userID, id uuid.UUID, withTrainings bool) error
	// ImportCycle saves a new cycle with its goals and trainings in one transaction.
	ImportCycle(ctx context.Context, userID uuid.UUID, cycle CycleInput, trainings []Input) (ImportedCycle, error)
}

// Service implements the planning usecases.
type Service struct {
	trainings TrainingRepository
	library   LibraryRepository
	cycles    CycleRepository
	catalog   Catalog
}

func NewService(trainings TrainingRepository, library LibraryRepository, cycles CycleRepository, catalog Catalog) *Service {
	if trainings == nil {
		panic("trainings cannot be nil")
	}
	if library == nil {
		panic("library cannot be nil")
	}
	if cycles == nil {
		panic("cycles cannot be nil")
	}
	if catalog == nil {
		panic("catalog cannot be nil")
	}
	return &Service{trainings: trainings, library: library, cycles: cycles, catalog: catalog}
}

// TrainingHeader is a training's own fields, as saved.
type TrainingHeader struct {
	ID          uuid.UUID
	CycleID     *uuid.UUID
	Name        string
	Description string
	Active      bool
	StartDate   time.Time
	EndDate     time.Time // the last Sunday
	Weeks       int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Training is a saved training with its sessions and schedule.
type Training struct {
	TrainingHeader
	Sessions []Session
	Schedule [][]Slot // one list per week
}

type Session struct {
	ID     uuid.UUID
	Name   string
	Notes  string
	Blocks []Block
}

type Block struct {
	ID   uuid.UUID
	Kind string
	Name string
	Sets *int32
	Amount
	Score
	Notes     string
	Exercises []Exercise
}

type Exercise struct {
	ExerciseID uuid.UUID
	Sets       *int32
	Amount
	Tempo string
	Notes string
}

// Slot is one planned session of a week.
type Slot struct {
	ID        uuid.UUID
	SessionID uuid.UUID
}

// CycleHeader is a cycle's own fields, as saved.
type CycleHeader struct {
	ID          uuid.UUID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     time.Time
	Weeks       int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ImportedCycle is what a cycle import created: the cycle and its trainings in the order sent.
type ImportedCycle struct {
	CycleHeader
	Trainings []TrainingHeader
}

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }
