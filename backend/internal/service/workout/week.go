package workout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Week is a week of training as the tracker shows it: the sessions the active training asks
// for, the workouts finished so far and the unit to do next.
type Week struct {
	Training   *repos.GetActiveTrainingRow  `json:"training"` // nil when no training is active
	Units      []repos.ListTrainingUnitsRow `json:"units"`    // the active training's, in order
	Done       []repos.ListWorkoutsRow      `json:"done"`     // finished in the week, any training
	NextUnitID *uuid.UUID                   `json:"next_unit_id"`
	Current    *Workout                     `json:"current"` // the workout in progress, if any
}

// Week returns the user's week [from, to); the client sends its bounds, so the week follows
// its time zone. Units are done in turn: the suggested one follows the unit of the last
// finished workout of the training (in any week), wrapping around to the first.
func (s *Service) Week(ctx context.Context, userID uuid.UUID, from, to time.Time) (Week, error) {
	done, err := s.List(ctx, userID, from, to)
	if err != nil {
		return Week{}, err
	}
	current, err := s.Current(ctx, userID)
	if err != nil {
		return Week{}, err
	}
	week := Week{Units: []repos.ListTrainingUnitsRow{}, Done: done, Current: current}

	training, err := s.q.GetActiveTraining(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return week, nil
	}
	if err != nil {
		return Week{}, err
	}
	week.Training = &training
	if week.Units, err = s.q.ListTrainingUnits(ctx, training.ID); err != nil {
		return Week{}, err
	}
	if len(week.Units) == 0 {
		return week, nil
	}

	last, err := s.q.GetLastUnitID(ctx, repos.GetLastUnitIDParams{UserID: userID, TrainingID: &training.ID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Week{}, err
	}
	next := 0 // the first unit, when none was done yet or the last one was removed
	if last != nil {
		if i := indexOf(week.Units, func(u repos.ListTrainingUnitsRow) bool { return u.ID == *last }); i >= 0 {
			next = (i + 1) % len(week.Units)
		}
	}
	week.NextUnitID = &week.Units[next].ID
	return week, nil
}
