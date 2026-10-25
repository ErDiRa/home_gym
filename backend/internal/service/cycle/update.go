package cycle

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

// Update validates a cycle and replaces its fields and goals in one transaction. Moving its period
// doesn't move its trainings.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in Input) (Cycle, error) {
	if err := Validate(&in); err != nil {
		return Cycle{}, err
	}
	var c Cycle
	err := s.inTx(ctx, func(q Queries) error {
		row, err := q.UpdateCycle(ctx, repos.UpdateCycleParams{
			ID: id, UserID: userID, Name: in.Name, Description: in.Description, StartDate: in.Start, Weeks: in.Weeks,
		})
		if err != nil {
			return err
		}
		if err := writeGoals(ctx, q, id, in.Goals); err != nil {
			return err
		}
		c, err = withGoalsAndTrainings(ctx, q, repos.GetCycleRow(row))
		return err
	})
	if err != nil {
		return Cycle{}, mapError(err)
	}
	return c, nil
}
