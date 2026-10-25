package training

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestUpdateKeepsIDs(t *testing.T) {
	f, reps, _ := newFake()
	kept, keptSlot := uuid.New(), uuid.New()
	f.knownSessions[kept] = true
	f.knownSlots[keptSlot] = true
	in := twoSessionInput(reps)
	in.Sessions[1].ID = &kept        // Session 2 exists, Session 1 is new
	in.Schedule[1][0].ID = &keptSlot // week 2's slot exists, moved here from elsewhere
	in.Active = true
	id, userID := uuid.New(), uuid.New()

	got, err := newTestService(f).Update(context.Background(), userID, id, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || !got.Active || got.StartDate.Time.Format(time.DateOnly) != "2026-03-02" || got.EndDate.Time.Format(time.DateOnly) != "2026-03-15" {
		t.Errorf("updated = %+v", got)
	}
	if f.updated[0].UserID != userID || !f.updated[0].Active {
		t.Errorf("updated %+v, want an active training of %v", f.updated[0], userID)
	}
	if !slices.Equal(f.keptSessions, []uuid.UUID{kept}) {
		t.Errorf("kept sessions = %v, want [%v]", f.keptSessions, kept)
	}
	if len(f.insertedSessions) != 1 || f.insertedSessions[0].Name != "Session 1" || f.insertedSessions[0].Position != 1 {
		t.Errorf("inserted sessions = %+v", f.insertedSessions)
	}
	if len(f.updatedSessions) != 1 || f.updatedSessions[0].ID != kept || f.updatedSessions[0].Position != 2 || f.updatedSessions[0].TrainingID != id {
		t.Errorf("updated sessions = %+v", f.updatedSessions)
	}
	// Slots: the kept one is updated in place, the others are inserted, all pointing at the
	// saved sessions.
	if !slices.Equal(f.keptSlots, []uuid.UUID{keptSlot}) {
		t.Errorf("kept slots = %v, want [%v]", f.keptSlots, keptSlot)
	}
	wantUpdated := repos.UpdateWeekSessionParams{ID: keptSlot, TrainingID: id, Week: 2, Position: 1, SessionID: kept}
	if len(f.updatedSlots) != 1 || f.updatedSlots[0] != wantUpdated {
		t.Errorf("updated slots = %+v, want %+v", f.updatedSlots, wantUpdated)
	}
	if len(f.insertedSlots) != 3 || f.insertedSlots[0].SessionID != f.sessionIDs[0] || f.insertedSlots[1].SessionID != kept {
		t.Errorf("inserted slots = %+v", f.insertedSlots)
	}
	// Blocks are cleared before they are re-inserted; slots of dropped sessions go first.
	if f.indexOf("DeleteTrainingBlocks") > f.indexOf("InsertSessionBlock") {
		t.Errorf("calls = %v, want DeleteTrainingBlocks first", f.calls)
	}
	if f.indexOf("DeleteWeekSessionsExcept") > f.indexOf("UpdateWeekSession") {
		t.Errorf("calls = %v, want DeleteWeekSessionsExcept before UpdateWeekSession", f.calls)
	}
}

func TestUpdateUnknownSession(t *testing.T) {
	f, reps, _ := newFake()
	foreign := uuid.New() // not in f.knownSessions: belongs to another training
	in := twoSessionInput(reps)
	in.Sessions[0].ID = &foreign

	_, err := newTestService(f).Update(context.Background(), uuid.New(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown session")
	if f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls end with %q, want rollback", f.calls[len(f.calls)-1])
	}
}

func TestUpdateUnknownSlot(t *testing.T) {
	f, reps, _ := newFake()
	foreign := uuid.New() // not in f.knownSlots: belongs to another training
	in := twoSessionInput(reps)
	in.Schedule[0][0].ID = &foreign

	_, err := newTestService(f).Update(context.Background(), uuid.New(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown slot in the schedule")
	if f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls end with %q, want rollback", f.calls[len(f.calls)-1])
	}
}

func TestUpdateMissingTraining(t *testing.T) {
	f, reps, _ := newFake()
	f.updateErr = pgx.ErrNoRows

	_, err := newTestService(f).Update(context.Background(), uuid.New(), uuid.New(), twoSessionInput(reps))
	wantError(t, err, service.KindNotFound, "training not found")
}

func TestUpdateIntoCycle(t *testing.T) {
	f, reps, _ := newFake()
	cycleID := uuid.New()
	f.ownedCycles[cycleID] = true
	in := twoSessionInput(reps)
	in.CycleID = &cycleID

	if _, err := newTestService(f).Update(context.Background(), uuid.New(), uuid.New(), in); err != nil {
		t.Fatal(err)
	}
	if f.updated[0].CycleID == nil || *f.updated[0].CycleID != cycleID {
		t.Errorf("updated into cycle %v, want %v", f.updated[0].CycleID, cycleID)
	}

	// Sending no cycle takes the training out of it.
	f2, reps2, _ := newFake()
	if _, err := newTestService(f2).Update(context.Background(), uuid.New(), uuid.New(), twoSessionInput(reps2)); err != nil {
		t.Fatal(err)
	}
	if f2.updated[0].CycleID != nil || f2.called("UserOwnsCycle") {
		t.Errorf("cycle = %v, calls = %v", f2.updated[0].CycleID, f2.calls)
	}
}

func TestUpdateIntoForeignCycle(t *testing.T) {
	f, reps, _ := newFake()
	in := twoSessionInput(reps)
	in.CycleID = ptr(uuid.New())

	_, err := newTestService(f).Update(context.Background(), uuid.New(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown cycle")
	if f.called("UpdateTraining") {
		t.Errorf("calls = %v", f.calls)
	}
}
