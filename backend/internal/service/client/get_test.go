package client

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

func TestGet(t *testing.T) {
	id := uuid.New()
	f := &fakeQueries{user: repos.GetUserByIDRow{ID: id, Email: "anna@example.com"}}
	got, err := newTestService(f).Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || len(got.Trainings) != 1 || got.Trainings[0].Name != "Strength" {
		t.Errorf("client = %+v", got)
	}

	f = &fakeQueries{userErr: pgx.ErrNoRows}
	_, err = newTestService(f).Get(context.Background(), id)
	wantError(t, err, service.KindNotFound, "client not found")
}
