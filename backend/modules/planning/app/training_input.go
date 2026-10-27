package app

import (
	"time"

	"github.com/google/uuid"
)

// Amount is the planned reps and load of a block or exercise. A value is nil when not set;
// a max turns it into a range (e.g. 8-10 reps). Units are rep_units / load_units ids.
type Amount struct {
	RepsValue *float64
	RepsMax   *float64
	RepsUnit  *uuid.UUID
	PerSide   bool
	LoadValue *float64
	LoadMax   *float64
	LoadUnit  *uuid.UUID
}

// clearReps and clearLoad empty the part of an amount that the block's kind keeps on the other
// side (block or exercises).
func (a *Amount) clearReps() { a.RepsValue, a.RepsMax, a.RepsUnit, a.PerSide = nil, nil, nil, false }
func (a *Amount) clearLoad() { a.LoadValue, a.LoadMax, a.LoadUnit = nil, nil, nil }

// ExerciseInput is one exercise in a block.
type ExerciseInput struct {
	ExerciseID uuid.UUID
	Sets       *int32 // nil when the block's kind shares sets
	Amount
	Tempo string // free text, e.g. '31X1'
	Notes string
} // Score is how a scored block (a WOD) is scored: the format (see the Score* constants) and its
// time in seconds, a time cap or the duration depending on the format, plus the interval of an
// EMOM. All nil on other blocks.
type Score struct {
	ScoreType       *string
	TimeSeconds     *int32
	IntervalSeconds *int32
}

// The formats of a scored block, as in the planning.session_blocks.score_type CHECK.
const (
	ScoreForTime     = "for_time"     // as fast as possible; time = optional cap
	ScoreAMRAP       = "amrap"        // as many rounds and reps as possible; time = duration
	ScoreMaxReps     = "max_reps"     // as many reps as possible; time = optional duration
	ScoreMaxCalories = "max_calories" // as many calories as possible; time = optional duration
	ScoreEMOM        = "emom"         // every interval on the interval; time = duration
)

type BlockInput struct {
	Kind string
	Name string
	Sets *int32 // nil unless the kind shares sets; the rounds of a WOD (optional)
	Amount
	Score
	Notes     string
	Exercises []ExerciseInput
}

// SessionInput is a session of a training, or a library template.
type SessionInput struct {
	// ID is set for a training's sessions that already exist, so they keep their id; new ones
	// have none. Ignored for templates (the id is in the URL).
	ID     *uuid.UUID
	Name   string
	Notes  string
	Blocks []BlockInput
}

// SlotInput is one planned session of a week. Session is the index of the session in
// Input.Sessions, as new sessions have no id yet.
type SlotInput struct {
	// ID is set for slots that already exist, so they keep their id (workouts point at them).
	ID      *uuid.UUID
	Session int
} // maxWeeks and maxSlots bound a training: at most a year, at most 7 sessions a week.
const (
	maxWeeks = 52
	maxSlots = 7
)

// Input is a whole training for Create and Update, the tree at once.
type Input struct {
	CycleID     *uuid.UUID // one of the user's cycles; nil = none
	Name        string
	Description string
	Active      bool
	StartDate   string // YYYY-MM-DD, a Monday
	Weeks       int32
	Sessions    []SessionInput
	Schedule    [][]SlotInput // one list per week, len == Weeks

	start time.Time // StartDate, parsed by validate
}

// Start is StartDate as parsed by validation.
func (in Input) Start() time.Time { return in.start }
