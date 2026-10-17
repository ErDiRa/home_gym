package client

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestDelete(t *testing.T) {
	if err := newTestService(&fakeQueries{deleteRows: 1}).Delete(context.Background(), uuid.New()); err != nil {
		t.Fatal(err)
	}
	err := newTestService(&fakeQueries{}).Delete(context.Background(), uuid.New())
	wantError(t, err, service.KindNotFound, "client not found")
}
