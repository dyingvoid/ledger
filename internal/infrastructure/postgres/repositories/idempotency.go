package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Idempotency) Claim(ctx context.Context, key string, expiresAt time.Time) (bool, error) {
	var claimedKey string
	err := r.executor(ctx).QueryRow(
		ctx,
		`INSERT INTO ledger.idempotency (key, status, result, error, expires_at)
		 VALUES ($1, $2, NULL, '', $3)
		 ON CONFLICT (key) DO UPDATE
		 SET status = EXCLUDED.status,
		     result = NULL,
		     error = '',
		     expires_at = EXCLUDED.expires_at,
		     updated_at = now()
		 WHERE ledger.idempotency.expires_at <= now()
		 RETURNING key`,
		key,
		string(idempotency.StatusInProgress),
		expiresAt,
	).Scan(&claimedKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claiming idempotency record: %w", err)
	}
	return true, nil
}

func (r *Idempotency) Complete(ctx context.Context, record idempotency.Record) error {
	var result any
	if record.Result != nil {
		result = string(record.Result)
	}

	tag, err := r.executor(ctx).Exec(
		ctx,
		`UPDATE ledger.idempotency
		 SET status = $2,
		     result = $3::jsonb,
		     error = $4,
		     expires_at = $5,
		     updated_at = now()
		 WHERE key = $1
		   AND status = $6`,
		record.Key,
		string(record.Status),
		result,
		record.Error,
		record.ExpiresAt,
		string(idempotency.StatusInProgress),
	)
	if err != nil {
		return fmt.Errorf("completing idempotency record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("completing idempotency record: %w", idempotency.ErrRecordNotFound)
	}
	return nil
}

func (r *Idempotency) executor(ctx context.Context) postgres.Executor {
	return postgres.ExecutorFrom(ctx, r.defaultExecutor)
}
