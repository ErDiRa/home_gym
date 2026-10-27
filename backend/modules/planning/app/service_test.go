package app

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
)

func TestCreateTraining(t *testing.T) {
	s, repo, cat := newTestService()
	got, err := s.CreateTraining(context.Background(), uuid.New(), twoSessions(cat))
	if err != nil {
		t.Fatal(err)
	}
	saved := repo.created[0]
	if saved.Name != "Strength" || !saved.start.Equal(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)) || got.Name != "Strength" {
		t.Errorf("saved %+v, got %+v; want the trimmed name and the parsed start", saved, got)
	}
}

func TestCreateTrainingInvalidWritesNothing(t *testing.T) {
	s, repo, cat := newTestService()
	in := twoSessions(cat)
	in.StartDate = "2026-03-03"
	_, err := s.CreateTraining(context.Background(), uuid.New(), in)
	wantError(t, err, errs.KindInvalid, "start_date must be a Monday")
	if len(repo.created) != 0 {
		t.Error("an invalid training was saved")
	}
}

// Only the user's live exercises can be planned: one the catalog doesn't know for the user, or a
// deleted one, is unknown.
func TestCreateTrainingUnknownExercise(t *testing.T) {
	for name, edit := range map[string]func(*fakeCatalog, *Input){
		"another user's": func(_ *fakeCatalog, in *Input) { in.Sessions[1].Blocks[0].Exercises[0].ExerciseID = uuid.New() },
		"deleted": func(cat *fakeCatalog, in *Input) {
			id := in.Sessions[1].Blocks[0].Exercises[0].ExerciseID
			cat.exercises[id] = catalog.Exercise{ID: id, Deleted: true}
		},
	} {
		t.Run(name, func(t *testing.T) {
			s, repo, cat := newTestService()
			in := twoSessions(cat)
			edit(cat, &in)
			_, err := s.CreateTraining(context.Background(), uuid.New(), in)
			wantError(t, err, errs.KindInvalid, "unknown exercise")
			if len(repo.created) != 0 {
				t.Error("saved anyway")
			}
		})
	}
}

func TestCreateTrainingRepositoryErrors(t *testing.T) {
	for _, tt := range []struct {
		err      error
		wantKind errs.Kind
		wantMsg  string
	}{
		{ErrUnknownCycle, errs.KindInvalid, "unknown cycle"},
		{ErrUnknownReference, errs.KindInvalid, "unknown exercise"},
	} {
		s, repo, cat := newTestService()
		repo.err = tt.err
		_, err := s.CreateTraining(context.Background(), uuid.New(), twoSessions(cat))
		wantError(t, err, tt.wantKind, tt.wantMsg)
	}
}

// An update may keep the ids of the training's own sessions and slots, and only those.
func TestUpdateTrainingKeptIDs(t *testing.T) {
	s1, slot := uuid.New(), uuid.New()
	current := Training{
		Sessions: []Session{{ID: s1, Name: "Session 1"}},
		Schedule: [][]Slot{{{ID: slot, SessionID: s1}}, {}},
	}
	tests := []struct {
		name    string
		edit    func(*Input)
		wantMsg string // empty: saved
	}{
		{"its own ids", func(in *Input) { in.Sessions[0].ID, in.Schedule[0][0].ID = &s1, &slot }, ""},
		{"another training's session", func(in *Input) { in.Sessions[0].ID = ptr(uuid.New()) }, "unknown session"},
		{"another training's slot", func(in *Input) { in.Schedule[1][0].ID = ptr(uuid.New()) }, "unknown slot in the schedule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, cat := newTestService()
			repo.current = current
			in := twoSessions(cat)
			tt.edit(&in)
			_, err := s.UpdateTraining(context.Background(), uuid.New(), uuid.New(), in)
			if tt.wantMsg == "" {
				if err != nil || len(repo.updated) != 1 || *repo.updated[0].Sessions[0].ID != s1 {
					t.Errorf("err = %v, updated %+v", err, repo.updated)
				}
				return
			}
			wantError(t, err, errs.KindInvalid, tt.wantMsg)
			if len(repo.updated) != 0 {
				t.Error("saved anyway")
			}
		})
	}

	s, repo, cat := newTestService()
	repo.err = ErrNotFound
	_, err := s.UpdateTraining(context.Background(), uuid.New(), uuid.New(), twoSessions(cat))
	wantError(t, err, errs.KindNotFound, "training not found")
}

func TestDuplicateTraining(t *testing.T) {
	s, repo, _ := newTestService()
	s1, s2, exercise, cycleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repo.current = Training{
		TrainingHeader: TrainingHeader{
			ID: uuid.New(), CycleID: &cycleID, Name: "Strength", Description: "Base", Active: true, Weeks: 2,
			StartDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		Sessions: []Session{
			{ID: s1, Name: "Session 1", Notes: "Heavy", Blocks: []Block{{Kind: "superset", Sets: ptr[int32](3),
				Exercises: []Exercise{{ExerciseID: exercise, Tempo: "31X1"}}}}},
			{ID: s2, Name: "Session 2"},
		},
		Schedule: [][]Slot{{{SessionID: s2}, {SessionID: s1}}, {{SessionID: s1}}},
	}

	if _, err := s.DuplicateTraining(context.Background(), uuid.New(), repo.current.ID); err != nil {
		t.Fatal(err)
	}
	copied := repo.created[0]
	// The copy is the next phase: as many weeks, from the Monday after the source ends, inactive,
	// in the same cycle.
	if copied.start.Format(time.DateOnly) != "2026-03-16" || copied.Weeks != 2 || copied.Active || *copied.CycleID != cycleID {
		t.Errorf("copy = %+v (start %v)", copied, copied.start)
	}
	if len(copied.Sessions) != 2 || copied.Sessions[0].ID != nil || copied.Sessions[0].Notes != "Heavy" ||
		copied.Sessions[0].Blocks[0].Exercises[0].Tempo != "31X1" {
		t.Errorf("sessions = %+v", copied.Sessions)
	}
	// The schedule points at the copied sessions by index.
	if !slices.Equal(copied.Schedule[0], []SlotInput{{Session: 1}, {Session: 0}}) || !slices.Equal(copied.Schedule[1], []SlotInput{{Session: 0}}) {
		t.Errorf("schedule = %+v", copied.Schedule)
	}

	repo.err = ErrNotFound
	_, err := s.DuplicateTraining(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, errs.KindNotFound, "training not found")
}

func TestLibrarySessions(t *testing.T) {
	s, repo, cat := newTestService()
	in := twoSessions(cat).Sessions[0]
	in.ID = ptr(uuid.New()) // ignored: the id is in the URL
	if _, err := s.CreateSession(context.Background(), uuid.New(), in); err != nil {
		t.Fatal(err)
	}
	if repo.sessions[0].ID != nil {
		t.Error("a template kept the id sent in the body")
	}

	in.Name = ""
	_, err := s.CreateSession(context.Background(), uuid.New(), in)
	wantError(t, err, errs.KindInvalid, "every session needs a name (max 100 characters)")

	repo.err = ErrNotFound
	_, err = s.CreateSession(context.Background(), uuid.New(), twoSessions(cat).Sessions[0])
	wantError(t, err, errs.KindNotFound, "session not found")
	wantError(t, s.DeleteSession(context.Background(), uuid.New(), uuid.New()), errs.KindNotFound, "session not found")
}

func TestCycles(t *testing.T) {
	s, repo, _ := newTestService()
	if _, err := s.CreateCycle(context.Background(), uuid.New(), CycleInput{Name: "Mobility", StartDate: "2026-03-03", Weeks: 8}); err == nil || len(repo.cycles) != 0 {
		t.Errorf("an invalid cycle: err = %v, saved %d", err, len(repo.cycles))
	}
	for _, withTrainings := range []bool{false, true} {
		if err := s.DeleteCycle(context.Background(), uuid.New(), uuid.New(), withTrainings); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(repo.withTrainings, []bool{false, true}) {
		t.Errorf("withTrainings = %v", repo.withTrainings)
	}
	repo.err = ErrNotFound
	err := s.UpdateCycle(context.Background(), uuid.New(), uuid.New(), CycleInput{Name: "Mobility", StartDate: "2026-03-02", Weeks: 8})
	wantError(t, err, errs.KindNotFound, "cycle not found")
}

// weeklyCycle is a 10-week cycle import with one one-week training per week, each starting on its
// week's Monday.
func weeklyCycle(cat *fakeCatalog) (CycleInput, []Input) {
	cycle := CycleInput{Name: "Master", StartDate: "2026-10-26", Weeks: 10, Goals: []GoalInput{{Title: "Peak in week 10"}}}
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	var trainings []Input
	for w := range 10 {
		t := twoSessions(cat)
		t.Name = fmt.Sprintf("Week %d of 10", w+1)
		t.StartDate = start.AddDate(0, 0, 7*w).Format(time.DateOnly)
		t.Weeks, t.Schedule = 1, t.Schedule[:1]
		trainings = append(trainings, t)
	}
	return cycle, trainings
}

func TestImportCycle(t *testing.T) {
	s, repo, cat := newTestService()
	cycle, trainings := weeklyCycle(cat)
	if _, err := s.ImportCycle(context.Background(), uuid.New(), cycle, trainings); err != nil {
		t.Fatal(err)
	}
	if len(repo.imported) != 10 || repo.cycles[0].Goals[0].Status != GoalOpen {
		t.Errorf("imported %d trainings, goals %+v", len(repo.imported), repo.cycles[0].Goals)
	}
}

// A broken training is reported with its week and name, and nothing is written.
func TestImportCycleNamesTheTraining(t *testing.T) {
	s, repo, cat := newTestService()
	cycle, trainings := weeklyCycle(cat)
	trainings[6].Sessions[1].Blocks[0].Kind = "foo"
	_, err := s.ImportCycle(context.Background(), uuid.New(), cycle, trainings)
	wantError(t, err, errs.KindInvalid, `Week 7 "Week 7 of 10": Session 2, block A: unknown block kind`)

	cycle, trainings = weeklyCycle(cat)
	trainings[2].Sessions[0].Blocks[0].Exercises[0].ExerciseID = uuid.New()
	_, err = s.ImportCycle(context.Background(), uuid.New(), cycle, trainings)
	wantError(t, err, errs.KindInvalid, `Week 3 "Week 3 of 10": unknown exercise`)
	if len(repo.cycles) != 0 {
		t.Error("written anyway")
	}
}

func TestImportCycleLimits(t *testing.T) {
	s, _, cat := newTestService()
	cycle, trainings := weeklyCycle(cat)
	for len(trainings) <= maxImportTrainings {
		trainings = append(trainings, trainings[0])
	}
	_, err := s.ImportCycle(context.Background(), uuid.New(), cycle, trainings)
	wantError(t, err, errs.KindInvalid, "a cycle import holds at most 60 trainings")

	// The cycle itself is validated first.
	cycle, trainings = weeklyCycle(cat)
	cycle.StartDate = "2026-10-27"
	_, err = s.ImportCycle(context.Background(), uuid.New(), cycle, trainings)
	wantError(t, err, errs.KindInvalid, "start_date must be a Monday")
}
