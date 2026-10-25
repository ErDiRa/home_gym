package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/account"
	"github.com/erdira/homegym/backend/internal/service/cycle"
	cyclerepos "github.com/erdira/homegym/backend/internal/service/cycle/repos"
	"github.com/erdira/homegym/backend/internal/service/exercise"
	exerciserepos "github.com/erdira/homegym/backend/internal/service/exercise/repos"
	"github.com/erdira/homegym/backend/internal/service/feedback"
	feedbackrepos "github.com/erdira/homegym/backend/internal/service/feedback/repos"
	"github.com/erdira/homegym/backend/internal/service/lookup"
	lookuprepos "github.com/erdira/homegym/backend/internal/service/lookup/repos"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
	"github.com/erdira/homegym/backend/internal/service/whoop"
	whooprepos "github.com/erdira/homegym/backend/internal/service/whoop/repos"
	"github.com/erdira/homegym/backend/internal/service/workout"
	workoutrepos "github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// The handlers depend on these interfaces rather than on the services in internal/service,
// so a service can be replaced by another implementation, e.g. a fake in handler tests.
// Failures the caller caused come back as *service.Error (see serviceError). The userID
// parameters are the signed-in user's: services act on that user's data only.

type TrainingService interface {
	List(ctx context.Context, userID uuid.UUID) ([]trainingrepos.ListTrainingsRow, error)
	Get(ctx context.Context, userID, id uuid.UUID) (training.Training, error)
	Create(ctx context.Context, userID uuid.UUID, in training.Input) (trainingrepos.CreateTrainingRow, error)
	Update(ctx context.Context, userID, id uuid.UUID, in training.Input) (trainingrepos.UpdateTrainingRow, error)
	Duplicate(ctx context.Context, userID, id uuid.UUID) (trainingrepos.CreateTrainingRow, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	ListSessions(ctx context.Context, userID uuid.UUID) ([]trainingrepos.ListLibrarySessionsRow, error)
	GetSession(ctx context.Context, userID, id uuid.UUID) (training.LibrarySession, error)
	CreateSession(ctx context.Context, userID uuid.UUID, in training.SessionInput) (training.LibrarySession, error)
	UpdateSession(ctx context.Context, userID, id uuid.UUID, in training.SessionInput) (training.LibrarySession, error)
	DeleteSession(ctx context.Context, userID, id uuid.UUID) error
	ImportCycle(ctx context.Context, userID uuid.UUID, in training.CycleImportInput) (training.ImportedCycle, error)
}

// CycleService manages the user's cycles; importing a whole cycle with its trainings is the
// TrainingService's (ImportCycle).
type CycleService interface {
	List(ctx context.Context, userID uuid.UUID) ([]cyclerepos.ListCyclesRow, error)
	Get(ctx context.Context, userID, id uuid.UUID) (cycle.Cycle, error)
	Create(ctx context.Context, userID uuid.UUID, in cycle.Input) (cycle.Cycle, error)
	Update(ctx context.Context, userID, id uuid.UUID, in cycle.Input) (cycle.Cycle, error)
	Delete(ctx context.Context, userID, id uuid.UUID, withTrainings bool) error
}

type ExerciseService interface {
	List(ctx context.Context, userID uuid.UUID) ([]exerciserepos.ListExercisesRow, error)
	Get(ctx context.Context, userID, id uuid.UUID) (exerciserepos.GetExerciseRow, error)
	Create(ctx context.Context, userID uuid.UUID, in exercise.Input) (exerciserepos.CreateExerciseRow, error)
	Update(ctx context.Context, userID, id uuid.UUID, in exercise.Input) (exerciserepos.UpdateExerciseRow, error)
}

type LookupService interface {
	Joints(ctx context.Context) ([]lookuprepos.ListJointsRow, error)
	MuscleGroups(ctx context.Context) ([]lookuprepos.ListMuscleGroupsRow, error)
	Focuses(ctx context.Context) ([]lookuprepos.ListFocusesRow, error)
	CreateFocus(ctx context.Context, in lookup.FocusInput) (lookuprepos.CreateFocusRow, error)
	BlockKinds(ctx context.Context) ([]lookuprepos.ListBlockKindsRow, error)
	RepUnits(ctx context.Context) ([]lookuprepos.ListRepUnitsRow, error)
	CreateRepUnit(ctx context.Context, in lookup.UnitInput) (lookuprepos.CreateRepUnitRow, error)
	LoadUnits(ctx context.Context) ([]lookuprepos.ListLoadUnitsRow, error)
	CreateLoadUnit(ctx context.Context, in lookup.UnitInput) (lookuprepos.CreateLoadUnitRow, error)
}

type FeedbackService interface {
	Rate(ctx context.Context, userID, exerciseID uuid.UUID, in feedback.RatingInput) (feedbackrepos.ExerciseFeedback, error)
	Ratings(ctx context.Context, userID uuid.UUID) ([]feedbackrepos.ListExerciseRatingsRow, error)
	Stats(ctx context.Context, userID uuid.UUID) ([]feedbackrepos.GetExerciseStatsRow, error)
}

type WorkoutService interface {
	Week(ctx context.Context, userID, trainingID uuid.UUID, monday, from, to time.Time) (workout.Week, error)
	List(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]workoutrepos.ListWorkoutsRow, error)
	Start(ctx context.Context, userID, weekSessionID uuid.UUID) (w workout.Workout, created bool, err error)
	Get(ctx context.Context, userID, id uuid.UUID) (workout.Workout, error)
	Finish(ctx context.Context, userID, id uuid.UUID, notes string, completedAt *time.Time) (workout.Workout, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	LogSet(ctx context.Context, userID, workoutID uuid.UUID, in workout.SetInput) (workoutrepos.WorkoutSet, error)
	DeleteSet(ctx context.Context, userID, workoutID, setID uuid.UUID) error
	LogResult(ctx context.Context, userID, workoutID uuid.UUID, in workout.ResultInput) (workoutrepos.WorkoutResult, error)
	DeleteResult(ctx context.Context, userID, workoutID, resultID uuid.UUID) error
}

// WhoopService connects the user's WHOOP account and links WHOOP activities to workouts.
type WhoopService interface {
	Configured() bool
	AuthURL(state string) string
	Connect(ctx context.Context, userID uuid.UUID, code string) error
	Connected(ctx context.Context, userID uuid.UUID) (bool, error)
	Disconnect(ctx context.Context, userID uuid.UUID) error
	Candidates(ctx context.Context, userID, workoutID uuid.UUID, from, to time.Time) ([]whoop.Candidate, error)
	Get(ctx context.Context, userID, workoutID uuid.UUID) (*whooprepos.WorkoutWhoopDatum, error)
	Link(ctx context.Context, userID, workoutID, whoopID uuid.UUID) (whooprepos.WorkoutWhoopDatum, error)
	Unlink(ctx context.Context, userID, workoutID uuid.UUID) error
}

type AccountService interface {
	Login(ctx context.Context, email, password string) (account.Account, error)
	Current(ctx context.Context, id uuid.UUID) (account.Account, error)
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Services are the services the handlers use.
type Services struct {
	Trainings TrainingService
	Cycles    CycleService
	Exercises ExerciseService
	Lookups   LookupService
	Feedback  FeedbackService
	Workouts  WorkoutService
	Whoop     WhoopService
	Accounts  AccountService
}

// The services in internal/service implement the interfaces.
var (
	_ TrainingService = (*training.Service)(nil)
	_ CycleService    = (*cycle.Service)(nil)
	_ ExerciseService = (*exercise.Service)(nil)
	_ LookupService   = (*lookup.Service)(nil)
	_ FeedbackService = (*feedback.Service)(nil)
	_ WorkoutService  = (*workout.Service)(nil)
	_ WhoopService    = (*whoop.Service)(nil)
	_ AccountService  = (*account.Service)(nil)
)
