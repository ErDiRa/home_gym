package cycle

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

// Delete soft-deletes one of the user's cycles. Its trainings are soft-deleted with it when
// withTrainings is set, else they stay and just leave the cycle; all in one transaction.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID, withTrainings bool) error {
	return s.inTx(ctx, func(q Queries) error {
		n, err := q.DeleteCycle(ctx, repos.DeleteCycleParams{ID: id, UserID: userID})
		if err != nil {
			return err
		}
		if n == 0 {
			return service.NotFound("cycle not found", nil)
		}
		// A soft delete doesn't fire the foreign key's ON DELETE SET NULL.
		if withTrainings {
			return q.DeleteCycleTrainings(ctx, id)
		}
		return q.ClearCycleFromTrainings(ctx, id)
	})
}
