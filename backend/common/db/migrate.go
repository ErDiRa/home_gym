// Package db holds the database plumbing the modules share: running a module's migrations in its
// own schema and running a function in a transaction.
package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// MigrateUp applies the migrations in dir of migrations (NNNN_name.up.sql) that the module's schema
// doesn't have yet. The schema is named after the module and created first; its version is kept in
// <schema>.schema_migrations, so every module migrates on its own. The migrations name their
// tables with the schema (identity.users), as nothing sets the search path for them.
func MigrateUp(ctx context.Context, schema string, pool *pgxpool.Pool, migrations fs.FS, dir string) error {
	if _, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{schema}.Sanitize()); err != nil {
		return fmt.Errorf("create schema %s: %w", schema, err)
	}
	src, err := iofs.New(migrations, dir)
	if err != nil {
		return fmt.Errorf("read %s migrations: %w", schema, err)
	}
	// Closing the *sql.DB (via m.Close) leaves the pool open.
	driver, err := pgxmigrate.WithInstance(stdlib.OpenDBFromPool(pool), &pgxmigrate.Config{
		SchemaName:      schema,
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("open %s migration driver: %w", schema, err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("init %s migrations: %w", schema, err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate %s: %w", schema, err)
	}
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("read %s migration version: %w", schema, err)
	}
	slog.InfoContext(ctx, "database migrated", "schema", schema, "version", version, "dirty", dirty)
	return nil
}
