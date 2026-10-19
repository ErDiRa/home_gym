package workout

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Week is a week of training as the tracker shows it: the active trainings to choose from, and
// for the chosen one the sessions it plans for the week; the workouts finished so far (the ones
// started from a slot mark it done) and the workout in progress, of any training.
type Week struct {
	Trainings  []repos.ListActiveTrainingsRow `json:"trainings"`   // the active ones, by name
	Training   *repos.ListActiveTrainingsRow  `json:"training"`    // the chosen one; nil when none is active
	WeekNumber *int32                         `json:"week_number"` // of the training, from 1; nil outside its weeks
	Planned    []repos.ListWeekPlanRow        `json:"planned"`     // the week's slots, in order
	Done       []repos.ListWorkoutsRow        `json:"done"`        // finished in the week, any training
	Current    *Workout                       `json:"current"`     // the workout in progress, if any
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
	current, err := s.Current(ctx, userID)
	if err != nil {
		return Week{}, err
	}
	trainings, err := s.q.ListActiveTrainings(ctx, userID)
	if err != nil {
		return Week{}, err
	}
	week := Week{Trainings: trainings, Planned: []repos.ListWeekPlanRow{}, Done: done, Current: current}
	if len(trainings) == 0 {
		return week, nil
	}
	training := &trainings[0]
	if i := indexOf(trainings, func(t repos.ListActiveTrainingsRow) bool { return t.ID == trainingID }); i >= 0 {
		training = &trainings[i]
	}
	week.Training = training

	// Whole days between the Mondays; rounded, as both are dates at midnight UTC.
	days := int32(monday.Sub(training.StartDate.Time).Round(24*time.Hour).Hours() / 24)
	if days < 0 || days >= training.Weeks*7 {
		return week, nil
	}
	number := days/7 + 1
	week.WeekNumber = &number
	if week.Planned, err = s.q.ListWeekPlan(ctx, repos.ListWeekPlanParams{TrainingID: training.ID, Week: number}); err != nil {
		return Week{}, err
	}
	return week, nil
}
