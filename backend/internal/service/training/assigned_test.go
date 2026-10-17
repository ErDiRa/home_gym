package training

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestAssigned(t *testing.T) {
	f, _, _ := newFake()
	id := uuid.New()
	f.assignedIDs = []uuid.UUID{id}
	f.training = repos.GetTrainingRow{ID: id, Name: "Strength", Active: true}
	f.units = []repos.ListTrainingUnitsRow{{ID: uuid.New(), Name: "Day A", Weekdays: []int16{1, 4}}}

	got, err := newTestService(f).Assigned(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != id || len(got[0].Units) != 1 || got[0].Units[0].Name != "Day A" {
		t.Errorf("assigned = %+v", got)
	}
}

func TestAssignedNone(t *testing.T) {
	f, _, _ := newFake()

	got, err := newTestService(f).Assigned(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	// An empty list, not nil, so it encodes as [].
	if got == nil || len(got) != 0 || f.called("GetTraining") {
		t.Errorf("assigned = %#v, calls = %v", got, f.calls)
	}
}
