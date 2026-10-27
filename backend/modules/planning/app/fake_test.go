package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
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

// fakeCatalog is the catalog's contract with the single and superset kinds, the units reps and kg,
// and the exercises in exercises (anyone's: the fake doesn't check the user).
type fakeCatalog struct {
	exercises map[uuid.UUID]catalog.Exercise
	reps, kg  uuid.UUID
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{exercises: map[uuid.UUID]catalog.Exercise{}, reps: uuid.New(), kg: uuid.New()}
}

// exercise adds a live exercise and returns its id.
func (c *fakeCatalog) exercise() uuid.UUID {
	id := uuid.New()
	c.exercises[id] = catalog.Exercise{ID: id, Name: "Squat", Type: "strength"}
	return id
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

func (c *fakeCatalog) BlockKinds(context.Context) ([]catalog.BlockKind, error) {
	return []catalog.BlockKind{
		{Code: "single", Name: "Single", MinExercises: 1},
		{Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
	}, nil
}

func (c *fakeCatalog) Units(context.Context) (catalog.Units, error) {
	return catalog.Units{
		Rep:  []catalog.Unit{{ID: c.reps, Name: "reps", MinValue: ptr(1.0)}},
		Load: []catalog.Unit{{ID: c.kg, Name: "kg", MinValue: ptr(0.0)}},
	}, nil
}

// fakeRepo implements the three repositories: it records what is saved and fails with err when it
// is set. current is the training UpdateTraining and Training load.
type fakeRepo struct {
	err     error
	current Training

	created       []Input
	updated       []Input
	deleted       []uuid.UUID
	sessions      []SessionInput
	cycles        []CycleInput
	withTrainings []bool
	imported      []Input
}

func (f *fakeRepo) CreateTraining(_ context.Context, _ uuid.UUID, in Input) (TrainingHeader, error) {
	if f.err != nil {
		return TrainingHeader{}, f.err
	}
	f.created = append(f.created, in)
	return TrainingHeader{ID: uuid.New(), Name: in.Name, StartDate: in.start, Weeks: in.Weeks}, nil
}

func (f *fakeRepo) UpdateTraining(ctx context.Context, _, id uuid.UUID, update func(context.Context, Training) (Input, error)) (TrainingHeader, error) {
	if f.err != nil {
		return TrainingHeader{}, f.err
	}
	in, err := update(ctx, f.current)
	if err != nil {
		return TrainingHeader{}, err
	}
	f.updated = append(f.updated, in)
	return TrainingHeader{ID: id, Name: in.Name}, nil
}

func (f *fakeRepo) Training(context.Context, uuid.UUID, uuid.UUID) (Training, error) {
	return f.current, f.err
}

func (f *fakeRepo) DeleteTraining(_ context.Context, _, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeRepo) CreateSession(_ context.Context, _ uuid.UUID, in SessionInput) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	f.sessions = append(f.sessions, in)
	return uuid.New(), nil
}

func (f *fakeRepo) UpdateSession(_ context.Context, _, _ uuid.UUID, in SessionInput) error {
	if f.err != nil {
		return f.err
	}
	f.sessions = append(f.sessions, in)
	return nil
}

func (f *fakeRepo) DeleteSession(context.Context, uuid.UUID, uuid.UUID) error { return f.err }

func (f *fakeRepo) CreateCycle(_ context.Context, _ uuid.UUID, in CycleInput) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	f.cycles = append(f.cycles, in)
	return uuid.New(), nil
}

func (f *fakeRepo) UpdateCycle(_ context.Context, _, _ uuid.UUID, in CycleInput) error {
	if f.err != nil {
		return f.err
	}
	f.cycles = append(f.cycles, in)
	return nil
}

func (f *fakeRepo) DeleteCycle(_ context.Context, _, _ uuid.UUID, withTrainings bool) error {
	if f.err != nil {
		return f.err
	}
	f.withTrainings = append(f.withTrainings, withTrainings)
	return nil
}

func (f *fakeRepo) ImportCycle(_ context.Context, _ uuid.UUID, cycle CycleInput, trainings []Input) (ImportedCycle, error) {
	if f.err != nil {
		return ImportedCycle{}, f.err
	}
	f.cycles = append(f.cycles, cycle)
	f.imported = trainings
	return ImportedCycle{CycleHeader: CycleHeader{ID: uuid.New(), Name: cycle.Name}}, nil
}

func newTestService() (*Service, *fakeRepo, *fakeCatalog) {
	repo, cat := &fakeRepo{}, newFakeCatalog()
	return NewService(repo, repo, repo, cat), repo, cat
}

// twoSessions is a valid 2-week training with two single-block sessions, both planned in week 1
// and the first again in week 2.
func twoSessions(cat *fakeCatalog) Input {
	session := func(name string) SessionInput {
		return SessionInput{Name: name, Blocks: []BlockInput{{Kind: "single", Exercises: []ExerciseInput{{
			ExerciseID: cat.exercise(), Sets: ptr[int32](3), Amount: Amount{RepsValue: ptr(5.0), RepsUnit: &cat.reps},
		}}}}}
	}
	return Input{
		Name: " Strength ", StartDate: "2026-03-02", Weeks: 2,
		Sessions: []SessionInput{session("Session 1"), session("Session 2")},
		Schedule: [][]SlotInput{{{Session: 0}, {Session: 1}}, {{Session: 0}}},
	}
}
