package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/core/accounts"
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

func (r *Currencies) Get(ctx context.Context, currency domain.Currency) (domain.CurrencyInfo, error) {
	info := domain.CurrencyInfo{Code: currency}
	err := r.executor(ctx).QueryRow(
		ctx,
		`SELECT ledger, asset_scale
		 FROM ledger.currency
		 WHERE code = $1`,
		string(currency),
	).Scan(
		&info.Ledger,
		&info.AssetScale,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CurrencyInfo{}, accounts.ErrCurrencyNotFound
		}
		return domain.CurrencyInfo{}, fmt.Errorf("getting currency: %w", err)
	}
	return info, nil
}

func (r *Currencies) executor(ctx context.Context) postgres.Executor {
	return postgres.ExecutorFrom(ctx, r.defaultExecutor)
}
