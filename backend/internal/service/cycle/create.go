package cycle

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

// Create validates a new cycle and saves it for the user with its goals in one transaction. It
// starts without trainings.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (Cycle, error) {
	if err := Validate(&in); err != nil {
		return Cycle{}, err
	}
	var c Cycle
	err := s.inTx(ctx, func(q Queries) error {
		row, err := q.CreateCycle(ctx, repos.CreateCycleParams{
			UserID: userID, Name: in.Name, Description: in.Description, StartDate: in.Start, Weeks: in.Weeks,
		})
		if err != nil {
			return err
		}
		if err := writeGoals(ctx, q, row.ID, in.Goals); err != nil {
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
