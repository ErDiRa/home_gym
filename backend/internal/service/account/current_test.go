package account

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestCurrent(t *testing.T) {
	f, userID := newFake()
	s := &Service{q: f}

	user, err := s.Current(context.Background(), userID)
	if err != nil || user.ID != userID || user.Email != "me@example.com" {
		t.Errorf("user = %+v, %v", user, err)
	}

	_, err = s.Current(context.Background(), uuid.New())
	wantError(t, err, service.KindNotFound, "account no longer exists")
}
