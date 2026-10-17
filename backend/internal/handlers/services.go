package handlers

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/account"
	"github.com/erdira/homegym/backend/internal/service/client"
	clientrepos "github.com/erdira/homegym/backend/internal/service/client/repos"
	"github.com/erdira/homegym/backend/internal/service/exercise"
	exerciserepos "github.com/erdira/homegym/backend/internal/service/exercise/repos"
	"github.com/erdira/homegym/backend/internal/service/feedback"
	feedbackrepos "github.com/erdira/homegym/backend/internal/service/feedback/repos"
	"github.com/erdira/homegym/backend/internal/service/lookup"
	lookuprepos "github.com/erdira/homegym/backend/internal/service/lookup/repos"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
)

// The handlers depend on these interfaces rather than on the services in internal/service,
// so a service can be replaced by another implementation, e.g. a fake in handler tests.
// Failures the caller caused come back as *service.Error (see serviceError).

type TrainingService interface {
	List(ctx context.Context) ([]trainingrepos.ListTrainingsRow, error)
	Get(ctx context.Context, id uuid.UUID) (training.Training, error)
	Create(ctx context.Context, in training.Input) (trainingrepos.CreateTrainingRow, error)
	Update(ctx context.Context, id uuid.UUID, in training.Input) (trainingrepos.UpdateTrainingRow, error)
	Duplicate(ctx context.Context, id, clientID uuid.UUID) (trainingrepos.CreateTrainingRow, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ClientService interface {
	List(ctx context.Context) ([]clientrepos.ListUsersRow, error)
	Get(ctx context.Context, id uuid.UUID) (client.Client, error)
	Create(ctx context.Context, in client.Input) (clientrepos.CreateUserRow, error)
	Update(ctx context.Context, id uuid.UUID, in client.Input) (clientrepos.UpdateUserRow, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ExerciseRatings(ctx context.Context, id uuid.UUID) ([]clientrepos.ListClientExerciseRatingsRow, error)
}

type ExerciseService interface {
	List(ctx context.Context) ([]exerciserepos.ListExercisesRow, error)
	Get(ctx context.Context, id uuid.UUID) (exerciserepos.GetExerciseRow, error)
	Create(ctx context.Context, in exercise.Input) (exerciserepos.CreateExerciseRow, error)
	Update(ctx context.Context, id uuid.UUID, in exercise.Input) (exerciserepos.UpdateExerciseRow, error)
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
	Stats(ctx context.Context, userID uuid.UUID) ([]feedbackrepos.GetExerciseStatsRow, error)
}

type AccountService interface {
	Login(ctx context.Context, email, password string) (account.Account, error)
	Current(ctx context.Context, role string, id uuid.UUID) (account.Account, error)
	IsAdmin(ctx context.Context, id uuid.UUID) (bool, error)
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Services are the services the handlers use.
type Services struct {
	Trainings TrainingService
	Clients   ClientService
	Exercises ExerciseService
	Lookups   LookupService
	Feedback  FeedbackService
	Accounts  AccountService
}

// The services in internal/service implement the interfaces.
var (
	_ TrainingService = (*training.Service)(nil)
	_ ClientService   = (*client.Service)(nil)
	_ ExerciseService = (*exercise.Service)(nil)
	_ LookupService   = (*lookup.Service)(nil)
	_ FeedbackService = (*feedback.Service)(nil)
	_ AccountService  = (*account.Service)(nil)
)
