package cycle

import (
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
)

// Validate trims and validates a cycle, in place: an empty goal status becomes open, and Start is
// set. It returns a service.Invalid error for the first problem.
func Validate(in *Input) error {
	in.Name, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if in.Name == "" || tooLong(in.Name, 100) {
		return service.Invalid("name is required (max 100 characters)")
	}
	if tooLong(in.Description, 2000) {
		return service.Invalid("description is too long (max 2000 characters)")
	}
	start, err := time.Parse(time.DateOnly, in.StartDate)
	if err != nil {
		return service.Invalid("start_date is required (YYYY-MM-DD)")
	}
	if start.Weekday() != time.Monday {
		return service.Invalid("start_date must be a Monday")
	}
	in.Start = pgtype.Date{Time: start, Valid: true}
	if in.Weeks < 1 || in.Weeks > maxWeeks {
		return service.Invalid(fmt.Sprintf("weeks must be 1-%d", maxWeeks))
	}
	if len(in.Goals) > maxGoals {
		return service.Invalid(fmt.Sprintf("a cycle has at most %d goals", maxGoals))
	}
	for i := range in.Goals {
		if msg := checkGoal(&in.Goals[i]); msg != "" {
			return service.Invalid(fmt.Sprintf("goal %d: %s", i+1, msg))
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
