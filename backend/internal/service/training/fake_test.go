package training

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func ptr[T any](v T) *T { return &v }

// fakeQueries is an in-memory Queries: it returns the canned rows and errors set on it and
// records the calls and the parameters of the writes.
type fakeQueries struct {
	calls []string

	// Canned results.
	training      repos.GetTrainingRow
	getErr        error
	sessions      []repos.ListTrainingSessionsRow
	blocks        []repos.ListTrainingBlocksRow
	exercises     []repos.ListTrainingExercisesRow
	slots         []repos.ListWeekSessionsRow
	template      *repos.GetLibrarySessionRow // nil: no template found
	tmplBlocks    []repos.ListSessionBlocksRow
	tmplExercises []repos.ListSessionExercisesRow
	kinds         []repos.ListBlockKindsRow
	repUnits      []repos.ListRepUnitsRow
	loadUnits     []repos.ListLoadUnitsRow
	knownSessions map[uuid.UUID]bool // session ids UpdateTrainingSession finds
	knownSlots    map[uuid.UUID]bool // slot ids UpdateWeekSession finds
	foreignExer   map[uuid.UUID]bool // exercise ids InsertSessionExercise doesn't find
	updateErr     error
	deleteRows    int64
	insertExerErr error

	// Recorded writes.
	gotTraining       []repos.GetTrainingParams
	created           []repos.CreateTrainingParams
	updated           []repos.UpdateTrainingParams
	keptSessions      []uuid.UUID
	insertedSessions  []repos.InsertSessionParams
	sessionIDs        []uuid.UUID // the ids InsertSession returned, in order
	updatedSessions   []repos.UpdateTrainingSessionParams
	keptSlots         []uuid.UUID
	insertedSlots     []repos.InsertWeekSessionParams
	updatedSlots      []repos.UpdateWeekSessionParams
	insertedBlocks    []repos.InsertSessionBlockParams
	insertedExercises []repos.InsertSessionExerciseParams
	updatedTemplates  []repos.UpdateLibrarySessionParams
	clearedBlocksOf   []uuid.UUID
}

// newFake returns a fake whose lookups hold a single and a superset kind, a "reps" unit and
// a "kg" unit.
func newFake() (*fakeQueries, uuid.UUID, uuid.UUID) {
	reps, kg := uuid.New(), uuid.New()
	return &fakeQueries{
		kinds: []repos.ListBlockKindsRow{
			{Code: "single", Name: "Single", MinExercises: 1},
			{Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
		},
		repUnits:      []repos.ListRepUnitsRow{{ID: reps, Name: "reps"}},
		loadUnits:     []repos.ListLoadUnitsRow{{ID: kg, Name: "kg"}},
		knownSessions: map[uuid.UUID]bool{},
		knownSlots:    map[uuid.UUID]bool{},
		foreignExer:   map[uuid.UUID]bool{},
	}, reps, kg
}

// newTestService runs transactions on the fake itself, recording begin/commit/rollback.
func newTestService(f *fakeQueries) *Service {
	return &Service{q: f, inTx: func(_ context.Context, fn func(Queries) error) error {
		f.calls = append(f.calls, "begin")
		err := fn(f)
		if err != nil {
			f.calls = append(f.calls, "rollback")
		} else {
			f.calls = append(f.calls, "commit")
		}
		return err
	}}
}

func (f *fakeQueries) called(name string) bool {
	for _, c := range f.calls {
		if c == name {
			return true
		}
	}
	return false
}

// indexOf returns the position of the first call to name, or -1.
func (f *fakeQueries) indexOf(name string) int {
	for i, c := range f.calls {
		if c == name {
			return i
		}
	}
	return -1
}

func (f *fakeQueries) ListTrainings(context.Context, uuid.UUID) ([]repos.ListTrainingsRow, error) {
	f.calls = append(f.calls, "ListTrainings")
	return []repos.ListTrainingsRow{}, nil
}

func (f *fakeQueries) GetTraining(_ context.Context, arg repos.GetTrainingParams) (repos.GetTrainingRow, error) {
	f.calls = append(f.calls, "GetTraining")
	f.gotTraining = append(f.gotTraining, arg)
	return f.training, f.getErr
}

func (f *fakeQueries) ListTrainingSessions(context.Context, uuid.UUID) ([]repos.ListTrainingSessionsRow, error) {
	f.calls = append(f.calls, "ListTrainingSessions")
	return f.sessions, nil
}

func (f *fakeQueries) ListTrainingBlocks(context.Context, uuid.UUID) ([]repos.ListTrainingBlocksRow, error) {
	f.calls = append(f.calls, "ListTrainingBlocks")
	return f.blocks, nil
}

func (f *fakeQueries) ListTrainingExercises(context.Context, uuid.UUID) ([]repos.ListTrainingExercisesRow, error) {
	f.calls = append(f.calls, "ListTrainingExercises")
	return f.exercises, nil
}

func (f *fakeQueries) ListWeekSessions(context.Context, uuid.UUID) ([]repos.ListWeekSessionsRow, error) {
	f.calls = append(f.calls, "ListWeekSessions")
	return f.slots, nil
}

// endDate is what the generated column computes: the last Sunday of the weeks.
func endDate(start pgtype.Date, weeks int32) pgtype.Date {
	return addDays(start, int(weeks)*7-1)
}

func (f *fakeQueries) CreateTraining(_ context.Context, arg repos.CreateTrainingParams) (repos.CreateTrainingRow, error) {
	f.calls = append(f.calls, "CreateTraining")
	f.created = append(f.created, arg)
	return repos.CreateTrainingRow{ID: uuid.New(), Name: arg.Name, Description: arg.Description, Active: arg.Active,
		StartDate: arg.StartDate, EndDate: endDate(arg.StartDate, arg.Weeks), Weeks: arg.Weeks}, nil
}

func (f *fakeQueries) UpdateTraining(_ context.Context, arg repos.UpdateTrainingParams) (repos.UpdateTrainingRow, error) {
	f.calls = append(f.calls, "UpdateTraining")
	if f.updateErr != nil {
		return repos.UpdateTrainingRow{}, f.updateErr
	}
	f.updated = append(f.updated, arg)
	return repos.UpdateTrainingRow{ID: arg.ID, Name: arg.Name, Description: arg.Description, Active: arg.Active,
		StartDate: arg.StartDate, EndDate: endDate(arg.StartDate, arg.Weeks), Weeks: arg.Weeks}, nil
}

func (f *fakeQueries) DeleteTraining(context.Context, repos.DeleteTrainingParams) (int64, error) {
	f.calls = append(f.calls, "DeleteTraining")
	return f.deleteRows, nil
}

func (f *fakeQueries) InsertSession(_ context.Context, arg repos.InsertSessionParams) (uuid.UUID, error) {
	f.calls = append(f.calls, "InsertSession")
	f.insertedSessions = append(f.insertedSessions, arg)
	id := uuid.New()
	f.sessionIDs = append(f.sessionIDs, id)
	return id, nil
}

func (f *fakeQueries) UpdateTrainingSession(_ context.Context, arg repos.UpdateTrainingSessionParams) (int64, error) {
	f.calls = append(f.calls, "UpdateTrainingSession")
	if !f.knownSessions[arg.ID] {
		return 0, nil
	}
	f.updatedSessions = append(f.updatedSessions, arg)
	return 1, nil
}

func (f *fakeQueries) DeleteTrainingSessionsExcept(_ context.Context, arg repos.DeleteTrainingSessionsExceptParams) error {
	f.calls = append(f.calls, "DeleteTrainingSessionsExcept")
	f.keptSessions = arg.KeepIds
	return nil
}

func (f *fakeQueries) DeleteTrainingBlocks(context.Context, uuid.UUID) error {
	f.calls = append(f.calls, "DeleteTrainingBlocks")
	return nil
}

func (f *fakeQueries) InsertWeekSession(_ context.Context, arg repos.InsertWeekSessionParams) error {
	f.calls = append(f.calls, "InsertWeekSession")
	f.insertedSlots = append(f.insertedSlots, arg)
	return nil
}

func (f *fakeQueries) UpdateWeekSession(_ context.Context, arg repos.UpdateWeekSessionParams) (int64, error) {
	f.calls = append(f.calls, "UpdateWeekSession")
	if !f.knownSlots[arg.ID] {
		return 0, nil
	}
	f.updatedSlots = append(f.updatedSlots, arg)
	return 1, nil
}

func (f *fakeQueries) DeleteWeekSessionsExcept(_ context.Context, arg repos.DeleteWeekSessionsExceptParams) error {
	f.calls = append(f.calls, "DeleteWeekSessionsExcept")
	f.keptSlots = arg.KeepIds
	return nil
}

func (f *fakeQueries) ListLibrarySessions(context.Context, uuid.UUID) ([]repos.ListLibrarySessionsRow, error) {
	f.calls = append(f.calls, "ListLibrarySessions")
	return []repos.ListLibrarySessionsRow{}, nil
}

func (f *fakeQueries) GetLibrarySession(_ context.Context, arg repos.GetLibrarySessionParams) (repos.GetLibrarySessionRow, error) {
	f.calls = append(f.calls, "GetLibrarySession")
	if f.template == nil {
		return repos.GetLibrarySessionRow{}, pgx.ErrNoRows
	}
	row := *f.template
	row.ID = arg.ID
	return row, nil
}

func (f *fakeQueries) ListSessionBlocks(context.Context, uuid.UUID) ([]repos.ListSessionBlocksRow, error) {
	f.calls = append(f.calls, "ListSessionBlocks")
	return f.tmplBlocks, nil
}

func (f *fakeQueries) ListSessionExercises(context.Context, uuid.UUID) ([]repos.ListSessionExercisesRow, error) {
	f.calls = append(f.calls, "ListSessionExercises")
	return f.tmplExercises, nil
}

func (f *fakeQueries) UpdateLibrarySession(_ context.Context, arg repos.UpdateLibrarySessionParams) (int64, error) {
	f.calls = append(f.calls, "UpdateLibrarySession")
	if f.template == nil {
		return 0, nil
	}
	f.updatedTemplates = append(f.updatedTemplates, arg)
	return 1, nil
}

func (f *fakeQueries) DeleteSessionBlocks(_ context.Context, sessionID uuid.UUID) error {
	f.calls = append(f.calls, "DeleteSessionBlocks")
	f.clearedBlocksOf = append(f.clearedBlocksOf, sessionID)
	return nil
}

func (f *fakeQueries) DeleteLibrarySession(context.Context, repos.DeleteLibrarySessionParams) (int64, error) {
	f.calls = append(f.calls, "DeleteLibrarySession")
	return f.deleteRows, nil
}

func (f *fakeQueries) InsertSessionBlock(_ context.Context, arg repos.InsertSessionBlockParams) (uuid.UUID, error) {
	f.calls = append(f.calls, "InsertSessionBlock")
	f.insertedBlocks = append(f.insertedBlocks, arg)
	return uuid.New(), nil
}

func (f *fakeQueries) InsertSessionExercise(_ context.Context, arg repos.InsertSessionExerciseParams) (int64, error) {
	f.calls = append(f.calls, "InsertSessionExercise")
	if f.insertExerErr != nil {
		return 0, f.insertExerErr
	}
	if f.foreignExer[arg.ExerciseID] {
		return 0, nil
	}
	f.insertedExercises = append(f.insertedExercises, arg)
	return 1, nil
}

func (f *fakeQueries) ListBlockKinds(context.Context) ([]repos.ListBlockKindsRow, error) {
	f.calls = append(f.calls, "ListBlockKinds")
	return f.kinds, nil
}

func (f *fakeQueries) ListRepUnits(context.Context) ([]repos.ListRepUnitsRow, error) {
	f.calls = append(f.calls, "ListRepUnits")
	return f.repUnits, nil
}

func (f *fakeQueries) ListLoadUnits(context.Context) ([]repos.ListLoadUnitsRow, error) {
	f.calls = append(f.calls, "ListLoadUnits")
	return f.loadUnits, nil
}

// wantError fails the test unless err is a *service.Error of kind with message msg.
func wantError(t *testing.T, err error, kind service.Kind, msg string) {
	t.Helper()
	var svcErr *service.Error
	if !errors.As(err, &svcErr) {
		t.Fatalf("err = %v, want a service error %q", err, msg)
	}
	if svcErr.Kind != kind || svcErr.Msg != msg {
		t.Fatalf("err = {kind %d, %q}, want {kind %d, %q}", svcErr.Kind, svcErr.Msg, kind, msg)
	}
}
