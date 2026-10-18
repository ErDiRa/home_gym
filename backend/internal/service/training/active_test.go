package training

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestActive(t *testing.T) {
	f, _, _ := newFake()
	id, userID := uuid.New(), uuid.New()
	f.activeID = &id
	f.training = repos.GetTrainingRow{ID: id, Name: "Strength", Active: true, SessionsPerWeek: 3}
	f.units = []repos.ListTrainingUnitsRow{{ID: uuid.New(), Name: "Day A"}}

	got, err := newTestService(f).Active(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != id || len(got.Units) != 1 || got.Units[0].Name != "Day A" {
		t.Errorf("active = %+v", got)
	}
	if f.gotTraining[0].UserID != userID {
		t.Errorf("loaded the training of %v, want %v", f.gotTraining[0].UserID, userID)
	}
}

func TestActiveNone(t *testing.T) {
	f, _, _ := newFake()

	got, err := newTestService(f).Active(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	// nil encodes as null: the tracker shows that there is nothing planned.
	if got != nil || f.called("GetTraining") {
		t.Errorf("active = %#v, calls = %v", got, f.calls)
	}
}
