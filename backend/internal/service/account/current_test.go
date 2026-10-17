package account

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service"
)

func TestCurrent(t *testing.T) {
	f, userID, adminID := newFake()
	s := &Service{q: f}

	user, err := s.Current(context.Background(), auth.RoleUser, userID)
	if err != nil || user.ID != userID || user.IsAdmin {
		t.Errorf("user = %+v, %v", user, err)
	}
	admin, err := s.Current(context.Background(), auth.RoleAdmin, adminID)
	if err != nil || admin.ID != adminID || !admin.IsAdmin {
		t.Errorf("admin = %+v, %v", admin, err)
	}

	// The role picks the table: an admin id with the user role is not found.
	_, err = s.Current(context.Background(), auth.RoleUser, adminID)
	wantError(t, err, service.KindNotFound, "account no longer exists")
	_, err = s.Current(context.Background(), auth.RoleAdmin, uuid.New())
	wantError(t, err, service.KindNotFound, "account no longer exists")
}
