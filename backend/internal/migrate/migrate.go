// Package migrate applies the embedded schema migrations (db/migrations) at startup.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/erdira/homegym/backend/db"
)

// Up applies the migrations the database doesn't have yet. A database created before the
// migrations were tracked (000001 loaded by Postgres' docker-entrypoint-initdb.d, as in dev and
// on the first production deploy) has no version but already has the schema of 000001, so it is
// baselined at version 1 first.
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	var initialised bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.users') IS NOT NULL").Scan(&initialised); err != nil {
		return fmt.Errorf("check schema: %w", err)
	}

	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	// Closing the *sql.DB (via m.Close) leaves the pool open.
	driver, err := pgxmigrate.WithInstance(stdlib.OpenDBFromPool(pool), &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("open migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
	}
	defer m.Close()

	_, _, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) && initialised {
		slog.InfoContext(ctx, "baselining database at migration 1")
		if err := m.Force(1); err != nil {
			return fmt.Errorf("baseline: %w", err)
		}
	} else if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read migration version: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: %w", err)
	}
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	slog.InfoContext(ctx, "database migrated", "version", version, "dirty", dirty)
	return nil
}
