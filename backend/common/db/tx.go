package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Beginner starts transactions; *pgxpool.Pool is one.
type Beginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// maxTxAttempts bounds how often UpdateInTx retries after a serialization failure.
const maxTxAttempts = 5

// UpdateInTx runs fn in a repeatable-read transaction, committing when it returns nil and rolling
// back otherwise. A repository loads what it changes and writes it back inside fn, so a concurrent
// change of the same rows makes the commit fail with a serialization failure; then the whole
// transaction (fn included) is retried, a few times with a short growing pause.
func UpdateInTx(ctx context.Context, db Beginner, fn func(ctx context.Context, tx pgx.Tx) error) error {
	pause := 5 * time.Millisecond
	for attempt := 1; ; attempt++ {
		err := inTx(ctx, db, fn)
		if err == nil || !isSerializationFailure(err) || attempt == maxTxAttempts {
			return err
		}
		select {
		case <-time.After(pause):
			pause *= 2
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		}
	}
}

func inTx(ctx context.Context, db Beginner, fn func(ctx context.Context, tx pgx.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, rbErr)
			}
			return
		}
		err = tx.Commit(ctx)
	}()
	return fn(ctx, tx)
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		(pgErr.Code == pgerrcode.SerializationFailure || pgErr.Code == pgerrcode.DeadlockDetected)
}
