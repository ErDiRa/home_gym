package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// checkCycle fails with "unknown cycle" unless cycleID is nil or one of the user's live cycles
// (the cycles themselves belong to the cycle service).
func checkCycle(ctx context.Context, q Queries, userID uuid.UUID, cycleID *uuid.UUID) error {
	if cycleID == nil {
		return nil
	}
	ok, err := q.UserOwnsCycle(ctx, repos.UserOwnsCycleParams{ID: *cycleID, UserID: userID})
	if err != nil {
		return err
	}
	if !ok {
		return service.Invalid("unknown cycle")
	}
	return nil
}
