// Package testdb gives integration tests the test database (TEST_DB_SOURCE, created by `task
// test:db`), migrated like the service does it, and users of their own in it.
package testdb

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend"
	identitydb "github.com/erdira/homegym/backend/modules/identity/adapters/db"
)

var (
	once    sync.Once
	pool    *pgxpool.Pool
	initErr error
)

// Pool returns the migrated test database, shared by all tests of the package.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	once.Do(func() {
		dbSource := os.Getenv("TEST_DB_SOURCE")
		if dbSource == "" {
			initErr = errNoSource
			return
		}
		ctx := context.Background()
		if pool, initErr = backend.NewPool(ctx, dbSource); initErr != nil {
			return
		}
		// Building the service migrates every module's schema.
		_, initErr = backend.New(ctx, pool, slog.New(slog.NewTextHandler(io.Discard, nil)), backend.Config{JWTSecret: "integration-test-secret"})
	})
	if initErr != nil {
		t.Fatal(initErr)
	}
	return pool
}

type sourceError string

func (e sourceError) Error() string { return string(e) }

const errNoSource = sourceError("TEST_DB_SOURCE is not set; run the integration tests with `task test:integration`")

// User creates a user of the test's own (random email) and returns its id.
func User(t testing.TB, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id, err := identitydb.NewUsersRepo(pool).CreateUser(context.Background(), "user-"+uuid.NewString()+"@example.com", "Test", "User", "not a bcrypt hash")
	if err != nil {
		t.Fatal(err)
	}
	return id
}
