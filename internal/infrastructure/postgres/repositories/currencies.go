package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
)

type CurrencyInfos struct {
	pool *pgxpool.Pool
}

func NewCurrencyInfos(pool *pgxpool.Pool) *CurrencyInfos {
	return &CurrencyInfos{pool: pool}
}

func (r *CurrencyInfos) Get(ctx context.Context, currency domain.Currency) (domain.CurrencyInfo, error) {
	var (
		info       domain.CurrencyInfo
		ledger     uint32
		assetScale int16
	)

	err := r.pool.QueryRow(
		ctx,
		`SELECT name, ledger, asset_scale
		 FROM ledger.currency_info
		 WHERE code = $1`,
		string(currency),
	).Scan(
		&info.Name,
		&ledger,
		&assetScale,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CurrencyInfo{}, fmt.Errorf("%w: %q", accounts.ErrCurrencyNotFound, currency)
	}
	if err != nil {
		return domain.CurrencyInfo{}, fmt.Errorf("getting currency info: %w", err)
	}

	info.Code = currency
	info.Ledger = ledger
	info.AssetScale = assetScale
	return info, nil
}

func (r *CurrencyInfos) Exists(ctx context.Context, currency domain.Currency) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM ledger.currency_info WHERE code = $1)`,
		string(currency),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking currency: %w", err)
	}
	return exists, nil
}
