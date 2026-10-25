package workout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Week is a week of training as the tracker shows it: the active trainings to choose from, and
// for the chosen one the sessions it plans for the week and its cycle; the workouts finished so
// far (the ones started from a slot mark it done), of any training.
type Week struct {
	Trainings  []repos.ListActiveTrainingsRow `json:"trainings"`   // the active ones to offer, by name
	Training   *repos.ListActiveTrainingsRow  `json:"training"`    // the chosen one; nil when none is active
	WeekNumber *int32                         `json:"week_number"` // of the training, from 1; nil outside its weeks
	Planned    []repos.ListWeekPlanRow        `json:"planned"`     // the week's slots, in order
	Done       []repos.ListWorkoutsRow        `json:"done"`        // finished in the week, any training
	// The cycle of the chosen training, with its goals and the week of the cycle; nil when the
	// training isn't part of one.
	Cycle *WeekCycle `json:"cycle"`
}

// WeekCycle is the cycle of the chosen training as the tracker shows it.
type WeekCycle struct {
	ID         uuid.UUID                     `json:"id"`
	Name       string                        `json:"name"`
	StartDate  pgtype.Date                   `json:"start_date"`
	EndDate    pgtype.Date                   `json:"end_date"`
	Weeks      int32                         `json:"weeks"`
	WeekNumber *int32                        `json:"week_number"` // of the cycle, from 1; nil outside its weeks
	Goals      []repos.ListWeekCycleGoalsRow `json:"goals"`       // in order
}

// Week returns the user's week starting on monday (a local calendar date); from and to are the
// bounds [from, to) of that week in the client's time zone, for the finished workouts.
// Trainings start on a Monday, so their weeks are calendar weeks. trainingID picks one of the
// active trainings; uuid.Nil or one that isn't active picks the first.
func (s *Service) Week(ctx context.Context, userID, trainingID uuid.UUID, monday, from, to time.Time) (Week, error) {
	if monday.IsZero() || monday.Weekday() != time.Monday {
		return Week{}, service.Invalid("date is required (YYYY-MM-DD), a Monday")
	}
	done, err := s.List(ctx, userID, from, to)
	if err != nil {
		return Week{}, err
	}
	trainings, err := s.q.ListActiveTrainings(ctx, userID)
	if err != nil {
		return Week{}, err
	}
	trainings = forWeek(trainings, monday)
	week := Week{Trainings: trainings, Planned: []repos.ListWeekPlanRow{}, Done: done}
	if len(trainings) == 0 {
		return week, nil
	}
	training := &trainings[0]
	if i := indexOf(trainings, func(t repos.ListActiveTrainingsRow) bool { return t.ID == trainingID }); i >= 0 {
		training = &trainings[i]
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
	if week.Planned, err = s.q.ListWeekPlan(ctx, repos.ListWeekPlanParams{TrainingID: training.ID, Week: *number}); err != nil {
		return Week{}, err
	}
	return week, nil
}

// forWeek leaves out the active trainings of a cycle that don't run in the week of monday: a
// cycle planned week by week has many active one-week trainings, and the tracker should offer
// this week's. Trainings without a cycle are always offered. If none would be left, all are kept,
// so the chosen one can still tell when it starts or ended.
func forWeek(trainings []repos.ListActiveTrainingsRow, monday time.Time) []repos.ListActiveTrainingsRow {
	kept := make([]repos.ListActiveTrainingsRow, 0, len(trainings))
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
	c, err := s.q.GetWeekCycle(ctx, repos.GetWeekCycleParams{ID: *cycleID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // deleted meanwhile
	}
	if err != nil {
		return nil, err
	}
	goals, err := s.q.ListWeekCycleGoals(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return &WeekCycle{
		ID: c.ID, Name: c.Name, StartDate: c.StartDate, EndDate: c.EndDate, Weeks: c.Weeks,
		WeekNumber: weekNumber(monday, c.StartDate, c.Weeks), Goals: goals,
	}, nil
}

// weekNumber is the week of a period of weeks from start (a Monday) that monday falls in, from
// 1; nil outside its weeks.
func weekNumber(monday time.Time, start pgtype.Date, weeks int32) *int32 {
	// Whole days between the Mondays; rounded, as both are dates at midnight UTC.
	days := int32(monday.Sub(start.Time).Round(24*time.Hour).Hours() / 24)
	if days < 0 || days >= weeks*7 {
		return nil
	}
	n := days/7 + 1
	return &n
}
