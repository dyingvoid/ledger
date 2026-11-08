package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type executorContextKey struct{}

func WithExecutor(ctx context.Context, executor Executor) context.Context {
	return context.WithValue(ctx, executorContextKey{}, executor)
}

func ExecutorFrom(ctx context.Context, fallback Executor) Executor {
	if executor, ok := ctx.Value(executorContextKey{}).(Executor); ok {
		return executor
	}
	return fallback
}

func txFrom(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(executorContextKey{}).(pgx.Tx)
	return tx, ok
}
