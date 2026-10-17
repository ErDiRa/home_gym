package exercise

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestGetMissing(t *testing.T) {
	_, err := (&Service{q: &fakeQueries{err: pgx.ErrNoRows}}).Get(context.Background(), uuid.New())
	wantError(t, err, service.KindNotFound, "exercise not found")
}
