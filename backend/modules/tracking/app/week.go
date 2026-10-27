package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
)

// Week is a week of training as the tracker shows it: the active trainings to choose from, and for
// the chosen one the sessions it plans for the week and its cycle; the workouts finished so far
// (the ones started from a slot mark it done), of any training.
type Week struct {
	Trainings  []planning.ActiveTraining // the active ones to offer, by name
	Training   *planning.ActiveTraining  // the chosen one; nil when none is active
	WeekNumber *int32                    // of the training, from 1; nil outside its weeks
	Planned    []planning.PlannedSlot    // the week's slots, in order
	Done       []WorkoutSummary          // finished in the week, any training
	// The cycle of the chosen training with the week of the cycle; nil when the training isn't
	// part of one.
	Cycle *WeekCycle
}

// WeekCycle is the cycle of the chosen training with the week of the cycle the week is.
type WeekCycle struct {
	planning.Cycle
	WeekNumber *int32 // of the cycle, from 1; nil outside its weeks
}

// Week returns the user's week starting on monday (a local calendar date); from and to are the
// bounds [from, to) of that week in the client's time zone, for the finished workouts. Trainings
// start on a Monday, so their weeks are calendar weeks. trainingID picks one of the active
// trainings; uuid.Nil or one that isn't active picks the first.
func (s *Service) Week(ctx context.Context, userID, trainingID uuid.UUID, monday, from, to time.Time) (Week, error) {
	if monday.IsZero() || monday.Weekday() != time.Monday {
		return Week{}, errs.Invalid("date is required (YYYY-MM-DD), a Monday")
	}
	if err := CheckPeriod(from, to); err != nil {
		return Week{}, err
	}
	done, err := s.workouts.FinishedWorkouts(ctx, userID, from, to)
	if err != nil {
		return Week{}, err
	}
	trainings, err := s.planning.ActiveTrainings(ctx, userID)
	if err != nil {
		return Week{}, err
	}
	trainings = forWeek(trainings, monday)
	week := Week{Trainings: trainings, Planned: []planning.PlannedSlot{}, Done: done}
	if len(trainings) == 0 {
		return week, nil
	}
	training := &trainings[0]
	for i := range trainings {
		if trainings[i].ID == trainingID {
			training = &trainings[i]
		}
	}
	week.Training = training
	if week.Cycle, err = s.weekCycle(ctx, userID, training.CycleID, monday); err != nil {
		return Week{}, err
	}

	number := weekNumber(monday, training.StartDate, training.Weeks)
	if number == nil {
		return week, nil
	}
	week.WeekNumber = number
	if week.Planned, err = s.planning.WeekPlan(ctx, userID, training.ID, *number); err != nil {
		return Week{}, err
	}
	return week, nil
}

// forWeek leaves out the active trainings of a cycle that don't run in the week of monday: a cycle
// planned week by week has many active one-week trainings, and the tracker should offer this
// week's. Trainings without a cycle are always offered. If none would be left, all are kept, so
// the chosen one can still tell when it starts or ended.
func forWeek(trainings []planning.ActiveTraining, monday time.Time) []planning.ActiveTraining {
	kept := make([]planning.ActiveTraining, 0, len(trainings))
	for _, t := range trainings {
		if t.CycleID == nil || weekNumber(monday, t.StartDate, t.Weeks) != nil {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return trainings
	}
	return kept
}

// weekCycle loads the cycle of the chosen training with its goals; nil without one.
func (s *Service) weekCycle(ctx context.Context, userID uuid.UUID, cycleID *uuid.UUID, monday time.Time) (*WeekCycle, error) {
	if cycleID == nil {
		return nil, nil
	}
	c, err := s.planning.Cycle(ctx, userID, *cycleID)
	if errors.Is(err, planning.ErrNotFound) {
		return nil, nil // deleted meanwhile
	}
	if err != nil {
		return nil, err
	}
	return &WeekCycle{Cycle: c, WeekNumber: weekNumber(monday, c.StartDate, c.Weeks)}, nil
}

// weekNumber is the week of a period of weeks from start (a Monday) that monday falls in, from 1;
// nil outside its weeks.
func weekNumber(monday, start time.Time, weeks int32) *int32 {
	// Whole days between the Mondays; rounded, as both are dates at midnight UTC.
	days := int32(monday.Sub(start).Round(24*time.Hour).Hours() / 24)
	if days < 0 || days >= weeks*7 {
		return nil
	}
	n := days/7 + 1
	return &n
}
