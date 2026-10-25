package training

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Duplicate copies one of the user's trainings, with its sessions and schedule. The copy starts
// inactive, and as the next phase: as many weeks, from the Monday after the source ends, in the
// same cycle (e.g. next week of a plan that arrives week by week).
func (s *Service) Duplicate(ctx context.Context, userID, id uuid.UUID) (repos.CreateTrainingRow, error) {
	// The source was validated when it was saved, so the copy isn't checked again.
	var training repos.CreateTrainingRow
	err := s.inTx(ctx, func(q Queries) error {
		source, err := load(ctx, q, userID, id)
		if err != nil {
			return err
		}
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: userID, CycleID: source.CycleID, Name: source.Name, Description: source.Description,
			StartDate: addDays(source.EndDate, 1), Weeks: source.Weeks, // the end is a Sunday
		})
		if err != nil {
			return err
		}
		sessions, schedule := copyPlan(source)
		ids, err := insertSessions(ctx, q, userID, training.ID, sessions)
		if err != nil {
			return err
		}
		return writeSchedule(ctx, q, training.ID, schedule, ids)
	})
	if err != nil {
		return repos.CreateTrainingRow{}, mapError(err)
	}
	return training, nil
}

// addDays moves a date by n days.
func addDays(d pgtype.Date, n int) pgtype.Date {
	return pgtype.Date{Time: d.Time.AddDate(0, 0, n), Valid: d.Valid}
}
