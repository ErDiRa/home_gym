package training

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Duplicate copies a training, with its units, blocks and exercises, for a client (the same
// or another one). The copy starts inactive, and as the next phase: its period, of the same
// length, starts the day after the source's ends, and the units' dates move along with it.
func (s *Service) Duplicate(ctx context.Context, id, clientID uuid.UUID) (repos.CreateTrainingRow, error) {
	// The source was validated when it was saved, so the copy isn't checked again.
	var training repos.CreateTrainingRow
	err := s.inTx(ctx, func(q Queries) error {
		source, err := load(ctx, q, id)
		if err != nil {
			return err
		}
		if err := checkClientExists(ctx, q, clientID); err != nil {
			return err
		}
		// Whole days, so the shift is the same in every time zone.
		shift := int(source.EndDate.Time.Sub(source.StartDate.Time).Hours()/24) + 1
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: clientID, Name: source.Name, Description: source.Description,
			StartDate: addDays(source.StartDate, shift), EndDate: addDays(source.EndDate, shift),
		})
		if err != nil {
			return err
		}
		units := copyUnits(source.Units)
		for i := range units {
			for j, d := range units[i].dates {
				units[i].dates[j] = addDays(d, shift)
			}
		}
		return insertUnits(ctx, q, training.ID, units)
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
