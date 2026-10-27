//go:build integration

package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	catalogdb "github.com/erdira/homegym/backend/modules/catalog/adapters/db"
	catalogmodule "github.com/erdira/homegym/backend/modules/catalog/api/module"
	catalogapp "github.com/erdira/homegym/backend/modules/catalog/app"
	planningdb "github.com/erdira/homegym/backend/modules/planning/adapters/db"
	"github.com/erdira/homegym/backend/modules/planning/app"
	"github.com/erdira/homegym/backend/tests/testdb"
)

// failingRepo writes what the app's update returns with a slot id of another training, so the
// repository fails after it has written the sessions and blocks.
type failingRepo struct{ *planningdb.Repo }

func (f failingRepo) UpdateTraining(ctx context.Context, userID, id uuid.UUID, update func(context.Context, app.Training) (app.Input, error)) (app.TrainingHeader, error) {
	return f.Repo.UpdateTraining(ctx, userID, id, func(ctx context.Context, current app.Training) (app.Input, error) {
		in, err := update(ctx, current)
		in.Schedule[0][0].ID = ptr(uuid.New())
		return in, err
	})
}

func ptr[T any](v T) *T { return &v }

// training is a valid 1-week training of the user with one session planned once.
func training(t *testing.T, userID uuid.UUID, name string) app.Input {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t)
	exercise, err := catalogapp.NewService(catalogdb.NewRepo(pool), catalogdb.NewRepo(pool), catalogdb.NewRepo(pool)).
		CreateExercise(ctx, userID, catalogapp.ExerciseInput{Name: "Squat", Type: "strength"})
	if err != nil {
		t.Fatal(err)
	}
	units, err := catalogmodule.New(pool).Units(ctx)
	if err != nil {
		t.Fatal(err)
	}
	reps := units.Rep[0].ID
	now := time.Now().UTC()
	monday := now.AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
	return app.Input{
		Name: name, StartDate: monday.Format(time.DateOnly), Weeks: 1,
		Sessions: []app.SessionInput{{Name: "Session 1", Blocks: []app.BlockInput{{Kind: "single", Exercises: []app.ExerciseInput{{
			ExerciseID: exercise.ID, Sets: ptr[int32](3), Amount: app.Amount{RepsValue: ptr(5.0), RepsUnit: &reps},
		}}}}}},
		Schedule: [][]app.SlotInput{{{Session: 0}}},
	}
}

// A training is saved and replaced in one transaction: an update failing part-way leaves it as it
// was.
func TestUpdateTrainingRollsBack(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	userID := testdb.User(t, pool)
	repo := planningdb.NewRepo(pool)
	catalog := catalogmodule.New(pool)

	header, err := app.NewService(repo, repo, repo, catalog).CreateTraining(ctx, userID, training(t, userID, "Before"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.Training(ctx, userID, header.ID)
	if err != nil {
		t.Fatal(err)
	}

	in := training(t, userID, "After")
	in.Sessions = append(in.Sessions, app.SessionInput{Name: "Session 2", Blocks: in.Sessions[0].Blocks})
	failing := app.NewService(failingRepo{repo}, repo, repo, catalog)
	if _, err := failing.UpdateTraining(ctx, userID, header.ID, in); err == nil {
		t.Fatal("the update succeeded with another training's slot")
	}

	after, err := repo.Training(ctx, userID, header.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Name != "Before" || len(after.Sessions) != 1 || after.Sessions[0].ID != before.Sessions[0].ID ||
		after.Schedule[0][0].ID != before.Schedule[0][0].ID || len(after.Sessions[0].Blocks) != 1 {
		t.Errorf("after a failed update = %+v, want it unchanged: %+v", after, before)
	}
}

// Another user's training is missing for the repository too.
func TestTrainingIsTheUsers(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	owner := testdb.User(t, pool)
	repo := planningdb.NewRepo(pool)
	header, err := app.NewService(repo, repo, repo, catalogmodule.New(pool)).CreateTraining(ctx, owner, training(t, owner, "Mine"))
	if err != nil {
		t.Fatal(err)
	}
	other := testdb.User(t, pool)
	if _, err := repo.Training(ctx, other, header.ID); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("another user's training: %v", err)
	}
	if err := repo.DeleteTraining(ctx, other, header.ID); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("deleting another user's training: %v", err)
	}
}
