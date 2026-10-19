package training

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestCreateSession(t *testing.T) {
	f, reps, _ := newFake()
	f.template = &repos.GetLibrarySessionRow{Name: "Session 1"}
	in := twoSessionInput(reps).Sessions[0]
	in.Name = " Session 1 "
	in.ID = ptr(uuid.New()) // ignored: templates are created, never moved
	userID := uuid.New()

	got, err := newTestService(f).CreateSession(context.Background(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	// A template belongs to no training.
	s := f.insertedSessions
	if len(s) != 1 || s[0].TrainingID != nil || s[0].UserID != userID || s[0].Name != "Session 1" {
		t.Errorf("sessions = %+v", s)
	}
	if len(f.insertedBlocks) != 1 || f.insertedBlocks[0].SessionID != f.sessionIDs[0] || len(f.insertedExercises) != 2 {
		t.Errorf("blocks = %+v, exercises = %+v", f.insertedBlocks, f.insertedExercises)
	}
	if got.ID != f.sessionIDs[0] || f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("got %+v, calls %v", got, f.calls)
	}
}

func TestCreateSessionInvalid(t *testing.T) {
	f, _, _ := newFake()
	_, err := newTestService(f).CreateSession(context.Background(), uuid.New(), SessionInput{Name: " "})
	wantError(t, err, service.KindInvalid, "every session needs a name (max 100 characters)")
	if f.called("begin") {
		t.Errorf("calls = %v, want no transaction", f.calls)
	}
}

func TestUpdateSessionRewritesBlocks(t *testing.T) {
	f, reps, _ := newFake()
	f.template = &repos.GetLibrarySessionRow{Name: "Session 1"}
	id, userID := uuid.New(), uuid.New()

	if _, err := newTestService(f).UpdateSession(context.Background(), userID, id, twoSessionInput(reps).Sessions[1]); err != nil {
		t.Fatal(err)
	}
	if len(f.updatedTemplates) != 1 || f.updatedTemplates[0] != (repos.UpdateLibrarySessionParams{ID: id, UserID: userID, Name: "Session 2"}) {
		t.Errorf("updated = %+v", f.updatedTemplates)
	}
	if len(f.clearedBlocksOf) != 1 || f.clearedBlocksOf[0] != id || f.indexOf("DeleteSessionBlocks") > f.indexOf("InsertSessionBlock") {
		t.Errorf("cleared %v, calls %v", f.clearedBlocksOf, f.calls)
	}
	if len(f.insertedBlocks) != 1 || f.insertedBlocks[0].SessionID != id {
		t.Errorf("blocks = %+v", f.insertedBlocks)
	}
}

func TestUpdateSessionMissing(t *testing.T) {
	f, reps, _ := newFake() // no template: unknown, deleted, a training's or another user's
	_, err := newTestService(f).UpdateSession(context.Background(), uuid.New(), uuid.New(), twoSessionInput(reps).Sessions[0])
	wantError(t, err, service.KindNotFound, "session not found")
	if f.called("DeleteSessionBlocks") || f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls = %v", f.calls)
	}
}

func TestGetSession(t *testing.T) {
	f, _, _ := newFake()
	id, blockID, squat := uuid.New(), uuid.New(), uuid.New()
	f.template = &repos.GetLibrarySessionRow{Name: "Session 1"}
	f.tmplBlocks = []repos.ListSessionBlocksRow{{ID: blockID, SessionID: id, Kind: "single"}}
	f.tmplExercises = []repos.ListSessionExercisesRow{{BlockID: blockID, ExerciseID: squat, Name: "Squat"}}

	got, err := newTestService(f).GetSession(context.Background(), uuid.New(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || len(got.Blocks) != 1 || len(got.Blocks[0].Exercises) != 1 || got.Blocks[0].Exercises[0].ExerciseID != squat {
		t.Errorf("session = %+v", got)
	}
}

func TestGetSessionMissing(t *testing.T) {
	f, _, _ := newFake()
	_, err := newTestService(f).GetSession(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "session not found")
}

func TestDeleteSession(t *testing.T) {
	f, _, _ := newFake()
	f.deleteRows = 1
	if err := newTestService(f).DeleteSession(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatal(err)
	}
	f.deleteRows = 0
	err := newTestService(f).DeleteSession(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "session not found")
}
