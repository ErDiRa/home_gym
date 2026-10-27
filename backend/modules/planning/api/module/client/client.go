// Package client is the planning module's contract: what other modules may read of the plans.
package client

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned for a slot or cycle that isn't the user's (or is gone).
var ErrNotFound = errors.New("not found")

type Planning interface {
	// ActiveTrainings returns the user's active trainings with their cycle, by name.
	ActiveTrainings(ctx context.Context, userID uuid.UUID) ([]ActiveTraining, error)
	// WeekPlan returns the slots one of the user's trainings plans for a week (from 1), in order.
	WeekPlan(ctx context.Context, userID, trainingID uuid.UUID, week int32) ([]PlannedSlot, error)
	// Slot returns a slot of the week plan of one of the user's live trainings, or ErrNotFound.
	Slot(ctx context.Context, userID, slotID uuid.UUID) (Slot, error)
	// Cycle returns one of the user's live cycles with its goals, or ErrNotFound.
	Cycle(ctx context.Context, userID, cycleID uuid.UUID) (Cycle, error)
}

// ActiveTraining is a training the tracker offers. Its weeks are calendar weeks from StartDate (a
// Monday) to EndDate (a Sunday).
type ActiveTraining struct {
	ID        uuid.UUID
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Weeks     int32
	CycleID   *uuid.UUID
	CycleName *string
}

// PlannedSlot is one session planned for a week.
type PlannedSlot struct {
	ID          uuid.UUID
	SessionID   uuid.UUID
	SessionName string
}

// Slot is a planned session with its training, for starting a workout of it.
type Slot struct {
	ID          uuid.UUID
	TrainingID  uuid.UUID
	SessionID   uuid.UUID
	SessionName string
}

// Cycle is a cycle's period and goals.
type Cycle struct {
	ID        uuid.UUID
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Weeks     int32
	Goals     []Goal // in order
}

type Goal struct {
	Title  string
	Target string
	Status string // open, achieved or missed
}
