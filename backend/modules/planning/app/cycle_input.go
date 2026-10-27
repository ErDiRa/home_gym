package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// The statuses of a goal, as in the planning.cycle_goals.status CHECK.
const (
	GoalOpen     = "open"
	GoalAchieved = "achieved"
	GoalMissed   = "missed"
)

// maxGoals bounds a cycle's goals (its weeks are bounded like a training's, maxWeeks).
const maxGoals = 20

// GoalInput is one goal of a cycle. Goals are rewritten on every save, in the order sent.
type GoalInput struct {
	ID     *uuid.UUID // ignored on save; the client sends back what it loaded
	Title  string     // required, e.g. "Better hips"
	Target string     // optional, something to check at the end
	Status string     // open (default when empty), achieved or missed
	Notes  string
}

// CycleInput is a whole cycle for CreateCycle and UpdateCycle, and the cycle of a cycle import.
type CycleInput struct {
	Name        string
	Description string
	StartDate   string // YYYY-MM-DD, a Monday
	Weeks       int32
	Goals       []GoalInput

	start time.Time // StartDate, parsed by validateCycle
}

// validateCycle trims and validates a cycle, in place: an empty goal status becomes open, and its
// start is set. It returns a errs.Invalid error for the first problem.
func validateCycle(in *CycleInput) error {
	in.Name, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if in.Name == "" || tooLong(in.Name, 100) {
		return errs.Invalid("name is required (max 100 characters)")
	}
	if tooLong(in.Description, 2000) {
		return errs.Invalid("description is too long (max 2000 characters)")
	}
	start, err := time.Parse(time.DateOnly, in.StartDate)
	if err != nil {
		return errs.Invalid("start_date is required (YYYY-MM-DD)")
	}
	if start.Weekday() != time.Monday {
		return errs.Invalid("start_date must be a Monday")
	}
	in.start = start
	if in.Weeks < 1 || in.Weeks > maxWeeks {
		return errs.Invalid(fmt.Sprintf("weeks must be 1-%d", maxWeeks))
	}
	if len(in.Goals) > maxGoals {
		return errs.Invalid(fmt.Sprintf("a cycle has at most %d goals", maxGoals))
	}
	for i := range in.Goals {
		if msg := checkGoal(&in.Goals[i]); msg != "" {
			return errs.Invalid(fmt.Sprintf("goal %d: %s", i+1, msg))
		}
	}
	return nil
}

// checkGoal trims a goal in place and describes its first problem, or returns "".
func checkGoal(g *GoalInput) string {
	g.Title, g.Target, g.Notes = strings.TrimSpace(g.Title), strings.TrimSpace(g.Target), strings.TrimSpace(g.Notes)
	g.Status = strings.TrimSpace(g.Status)
	if g.Status == "" {
		g.Status = GoalOpen
	}
	switch {
	case g.Title == "" || tooLong(g.Title, 200):
		return "a title is required (max 200 characters)"
	case tooLong(g.Target, 200):
		return "the target is too long (max 200 characters)"
	case tooLong(g.Notes, 2000):
		return "the notes are too long (max 2000 characters)"
	case g.Status != GoalOpen && g.Status != GoalAchieved && g.Status != GoalMissed:
		return "the status must be open, achieved or missed"
	}
	return ""
}

// Start is StartDate as parsed by validation.
func (in CycleInput) Start() time.Time { return in.start }
