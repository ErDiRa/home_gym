package training

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestDelete(t *testing.T) {
	f, _, _ := newFake()
	f.deleteRows = 1
	if err := newTestService(f).Delete(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatal(err)
	}

	f.deleteRows = 0
	err := newTestService(f).Delete(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "training not found")
}
