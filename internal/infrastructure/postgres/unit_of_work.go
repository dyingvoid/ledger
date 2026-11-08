package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if parent, ok := txFrom(ctx); ok {
		nested, err := parent.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning nested transaction: %w", err)
		}
		return runInTransaction(ctx, nested, fn)
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	return runInTransaction(ctx, tx, fn)
}

func runInTransaction(ctx context.Context, tx pgx.Tx, fn func(ctx context.Context) error) (err error) {
	committed := false
	defer func() {
		if committed {
			return
		}
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = fmt.Errorf("rolling back transaction: %w", rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()

	if err = fn(WithExecutor(ctx, tx)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	committed = true
	return nil
}
