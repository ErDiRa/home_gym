package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
)

func ptr[T any](v T) *T { return &v }

// wantError fails the test unless err is a *errs.Error of kind with message msg.
func wantError(t *testing.T, err error, kind errs.Kind, msg string) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %v, want an errs.Error %q", err, msg)
	}
	if e.Kind != kind || e.Msg != msg {
		t.Fatalf("err = {kind %d, %q}, want {kind %d, %q}", e.Kind, e.Msg, kind, msg)
	}
}

// fakeRepo implements both repositories: workout is the user's only workout, sets and results
// record what is logged.
type fakeRepo struct {
	workout    uuid.UUID
	inProgress bool // CreateWorkout finds the slot in progress
	created    []planning.Slot
	finished   []string // notes
	done       []WorkoutSummary
	sets       []SetInput
	results    []ResultInput
	maxes      []MaxInput
	ownedSet   uuid.UUID
}

func (f *fakeRepo) CreateWorkout(_ context.Context, _ uuid.UUID, slot planning.Slot) (uuid.UUID, error) {
	if f.inProgress {
		return uuid.Nil, ErrSlotInProgress
	}
	f.created = append(f.created, slot)
	return uuid.New(), nil
}

func (f *fakeRepo) SlotWorkoutInProgress(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
	return f.workout, nil
}

func (f *fakeRepo) FinishWorkout(_ context.Context, _, id uuid.UUID, notes string, _ *time.Time) error {
	if id != f.workout {
		return ErrNotFound
	}
	f.finished = append(f.finished, notes)
	return nil
}

func (f *fakeRepo) DeleteWorkout(_ context.Context, _, id uuid.UUID) error {
	if id != f.workout {
		return ErrNotFound
	}
	return nil
}

func (f *fakeRepo) FinishedWorkouts(context.Context, uuid.UUID, time.Time, time.Time) ([]WorkoutSummary, error) {
	return f.done, nil
}

func (f *fakeRepo) UserOwnsWorkout(_ context.Context, _, id uuid.UUID) (bool, error) {
	return id == f.workout, nil
}

func (f *fakeRepo) LogSet(_ context.Context, workoutID uuid.UUID, in SetInput) (LoggedSet, error) {
	f.sets = append(f.sets, in)
	return LoggedSet{ID: uuid.New(), WorkoutID: workoutID, ExerciseID: in.ExerciseID}, nil
}

func (f *fakeRepo) DeleteSet(context.Context, uuid.UUID, uuid.UUID) error { return ErrNotFound }

func (f *fakeRepo) LogResult(_ context.Context, workoutID uuid.UUID, in ResultInput) (Result, error) {
	f.results = append(f.results, in)
	return Result{ID: uuid.New(), WorkoutID: workoutID, BlockPosition: in.BlockPosition}, nil
}

func (f *fakeRepo) DeleteResult(context.Context, uuid.UUID, uuid.UUID) error { return ErrNotFound }

func (f *fakeRepo) CreateMax(_ context.Context, _, _ uuid.UUID, in MaxInput, _ time.Time) (uuid.UUID, error) {
	f.maxes = append(f.maxes, in)
	return uuid.New(), nil
}

func (f *fakeRepo) UserOwnsSet(_ context.Context, _, setID, _ uuid.UUID) (bool, error) {
	return setID == f.ownedSet, nil
}

func (f *fakeRepo) DeleteMax(context.Context, uuid.UUID, uuid.UUID) error { return ErrNotFound }

// fakePlanning is the planning contract over fixed trainings, slots and cycles.
type fakePlanning struct {
	trainings []planning.ActiveTraining
	plans     map[uuid.UUID]map[int32][]planning.PlannedSlot // by training and week
	slot      planning.Slot
	cycles    map[uuid.UUID]planning.Cycle
}

func (p *fakePlanning) ActiveTrainings(context.Context, uuid.UUID) ([]planning.ActiveTraining, error) {
	return append([]planning.ActiveTraining{}, p.trainings...), nil
}

func (p *fakePlanning) WeekPlan(_ context.Context, _, trainingID uuid.UUID, week int32) ([]planning.PlannedSlot, error) {
	return p.plans[trainingID][week], nil
}

func (p *fakePlanning) Slot(_ context.Context, _, slotID uuid.UUID) (planning.Slot, error) {
	if slotID != p.slot.ID {
		return planning.Slot{}, planning.ErrNotFound
	}
	return p.slot, nil
}

func (p *fakePlanning) Cycle(_ context.Context, _, cycleID uuid.UUID) (planning.Cycle, error) {
	c, ok := p.cycles[cycleID]
	if !ok {
		return planning.Cycle{}, planning.ErrNotFound
	}
	return c, nil
}

// fakeCatalog is the catalog contract with the rep units reps and RPE-like "rpe" (1-10), the load
// units kg (a weight unit) and "% 1RM", and the exercises in exercises.
type fakeCatalog struct {
	exercises        map[uuid.UUID]catalog.Exercise
	reps, rpe        uuid.UUID
	kg, percentOfMax uuid.UUID
}

func (c *fakeCatalog) Exercises(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error) {
	out := map[uuid.UUID]catalog.Exercise{}
	for _, id := range ids {
		if e, ok := c.exercises[id]; ok {
			out[id] = e
		}
	}
	return out, nil
}

func (c *fakeCatalog) Units(context.Context) (catalog.Units, error) {
	return catalog.Units{
		Rep: []catalog.Unit{{ID: c.reps, Name: "reps", MinValue: ptr(1.0)}, {ID: c.rpe, Name: "rpe", MinValue: ptr(1.0), MaxValue: ptr(10.0)}},
		Load: []catalog.Unit{
			{ID: c.kg, Name: "kg", MinValue: ptr(0.0), KgPerUnit: ptr(1.0)},
			{ID: c.percentOfMax, Name: "% 1RM", PercentOfMax: true},
		},
	}, nil
}

type fixture struct {
	s        *Service
	repo     *fakeRepo
	planning *fakePlanning
	catalog  *fakeCatalog
	exercise uuid.UUID // a live exercise of the user's
}

func newFixture() fixture {
	exercise := uuid.New()
	f := fixture{
		repo:     &fakeRepo{workout: uuid.New()},
		planning: &fakePlanning{plans: map[uuid.UUID]map[int32][]planning.PlannedSlot{}, cycles: map[uuid.UUID]planning.Cycle{}},
		catalog: &fakeCatalog{
			exercises: map[uuid.UUID]catalog.Exercise{exercise: {ID: exercise, Name: "Squat"}},
			reps:      uuid.New(), rpe: uuid.New(), kg: uuid.New(), percentOfMax: uuid.New(),
		},
		exercise: exercise,
	}
	f.s = NewService(f.repo, f.repo, f.planning, f.catalog)
	f.s.now = func() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }
	return f
}
