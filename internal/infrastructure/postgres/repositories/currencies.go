package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
)

type Currencies struct {
	defaultExecutor postgres.Executor
}

func NewCurrencies(pool *pgxpool.Pool) *Currencies {
	return &Currencies{defaultExecutor: pool}
}

func (r *Currencies) Exists(ctx context.Context, currency domain.Currency) (bool, error) {
	var exists bool
	err := r.executor(ctx).QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM ledger.currency WHERE code = $1)`,
		string(currency),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking currency: %w", err)
	}
	return exists, nil
}

func (r *Currencies) executor(ctx context.Context) postgres.Executor {
	return postgres.ExecutorFrom(ctx, r.defaultExecutor)
}
