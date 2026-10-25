package cycle

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

// Deleting a cycle keeps its trainings by default: they just leave the cycle.
func TestDeleteKeepsTrainings(t *testing.T) {
	f := &fakeQueries{}
	f.deleteRows = 1
	id := uuid.New()

	if err := newTestService(f).Delete(context.Background(), uuid.New(), id, false); err != nil {
		t.Fatal(err)
	}
	if len(f.clearedCycleOf) != 1 || f.clearedCycleOf[0] != id || f.called("DeleteCycleTrainings") {
		t.Errorf("calls = %v", f.calls)
	}
	if f.calls[0] != "begin" || f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("calls = %v, want one transaction", f.calls)
	}
}

func TestDeleteWithTrainings(t *testing.T) {
	f := &fakeQueries{}
	f.deleteRows = 1
	id := uuid.New()

	if err := newTestService(f).Delete(context.Background(), uuid.New(), id, true); err != nil {
		t.Fatal(err)
	}
	if len(f.deletedTrainingsOf) != 1 || f.deletedTrainingsOf[0] != id || f.called("ClearCycleFromTrainings") {
		t.Errorf("calls = %v", f.calls)
	}
	if f.calls[0] != "begin" || f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("calls = %v, want one transaction", f.calls)
	}
}

func TestDeleteNotFound(t *testing.T) {
	f := &fakeQueries{}

	err := newTestService(f).Delete(context.Background(), uuid.New(), uuid.New(), true)
	wantError(t, err, service.KindNotFound, "cycle not found")
	if f.called("DeleteCycleTrainings") || f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls = %v", f.calls)
	}
}
