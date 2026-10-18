package training

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Duplicate copies one of the user's trainings, with its units, blocks and exercises. The copy
// starts inactive, and as the next phase: its period, of the same length, starts the day after
// the source's ends.
func (s *Service) Duplicate(ctx context.Context, userID, id uuid.UUID) (repos.CreateTrainingRow, error) {
	// The source was validated when it was saved, so the copy isn't checked again.
	var training repos.CreateTrainingRow
	err := s.inTx(ctx, func(q Queries) error {
		source, err := load(ctx, q, userID, id)
		if err != nil {
			return err
		}
		// Whole days, so the shift is the same in every time zone.
		shift := int(source.EndDate.Time.Sub(source.StartDate.Time).Hours()/24) + 1
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: userID, Name: source.Name, Description: source.Description,
			StartDate: addDays(source.StartDate, shift), EndDate: addDays(source.EndDate, shift),
			SessionsPerWeek: source.SessionsPerWeek,
		})
		if err != nil {
			return err
		}
		return insertUnits(ctx, q, userID, training.ID, copyUnits(source.Units))
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
