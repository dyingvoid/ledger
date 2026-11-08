package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
)

type Idempotency struct {
	defaultExecutor postgres.Executor
}

func NewIdempotency(pool *pgxpool.Pool) *Idempotency {
	return &Idempotency{defaultExecutor: pool}
}

func (r *Idempotency) Lock(ctx context.Context, key string) error {
	_, err := r.executor(ctx).Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		key,
	)
	if err != nil {
		return fmt.Errorf("acquiring idempotency lock: %w", err)
	}
	return nil
}

func (r *Idempotency) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	var (
		record idempotency.Record
		status string
	)

	err := r.executor(ctx).QueryRow(
		ctx,
		`SELECT key, status, COALESCE(result, 'null'::jsonb), error, expires_at
		 FROM ledger.idempotency
		 WHERE key = $1
		   AND expires_at > now()`,
		key,
	).Scan(&record.Key, &status, &record.Result, &record.Error, &record.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return idempotency.Record{}, false, nil
	}
	if err != nil {
		return idempotency.Record{}, false, fmt.Errorf("getting idempotency record: %w", err)
	}

	record.Status = idempotency.Status(status)
	return record, true, nil
}

func (r *Idempotency) Save(ctx context.Context, record idempotency.Record) error {
	var result any
	if record.Result != nil {
		result = string(record.Result)
	}

	_, err := r.executor(ctx).Exec(
		ctx,
		`INSERT INTO ledger.idempotency (key, status, result, error, expires_at)
		 VALUES ($1, $2, $3::jsonb, $4, $5)
		 ON CONFLICT (key) DO UPDATE
		 SET status = EXCLUDED.status,
		     result = EXCLUDED.result,
		     error = EXCLUDED.error,
		     expires_at = EXCLUDED.expires_at,
		     updated_at = now()`,
		record.Key,
		string(record.Status),
		result,
		record.Error,
		record.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("saving idempotency record: %w", err)
	}
	return nil
}

func (r *Idempotency) executor(ctx context.Context) postgres.Executor {
	return postgres.ExecutorFrom(ctx, r.defaultExecutor)
}
