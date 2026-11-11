package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/currencies"
	"github.com/dyingvoid/ledger/internal/domain"
)

type CurrencyInfos struct {
	pool *pgxpool.Pool
}

func NewCurrencyInfos(pool *pgxpool.Pool) *CurrencyInfos {
	return &CurrencyInfos{pool: pool}
}

func (r *CurrencyInfos) Create(
	ctx context.Context,
	currency domain.CurrencyInfo,
) (domain.CurrencyInfo, error) {
	created, err := scanCurrency(r.pool.QueryRow(
		ctx,
		`INSERT INTO ledger.currency_info (code, name, ledger, asset_scale)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (code) DO NOTHING
		 RETURNING code, name, ledger, asset_scale`,
		currency.Code,
		currency.Name,
		currency.Ledger,
		currency.AssetScale,
	))
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.CurrencyInfo{}, currencyError(err)
	}

	existing, err := r.Get(ctx, currency.Code)
	if err != nil {
		return domain.CurrencyInfo{}, err
	}
	if !sameCurrency(existing, currency) {
		return domain.CurrencyInfo{}, fmt.Errorf("%w: %s", currencies.ErrCurrencyConflict, currency.Code)
	}
	return existing, nil
}

func (r *CurrencyInfos) Get(ctx context.Context, currency string) (domain.CurrencyInfo, error) {
	info, err := scanCurrency(r.pool.QueryRow(
		ctx,
		`SELECT code, name, ledger, asset_scale
		 FROM ledger.currency_info
		 WHERE code = $1`,
		currency,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CurrencyInfo{}, fmt.Errorf("%w: %q", accounts.ErrCurrencyNotFound, currency)
	}
	if err != nil {
		return domain.CurrencyInfo{}, fmt.Errorf("getting currency info: %w", err)
	}
	return info, nil
}

// TODO: doesn't seem to be used at all
func (r *CurrencyInfos) Exists(ctx context.Context, currency string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM ledger.currency_info WHERE code = $1)`,
		currency,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking currency: %w", err)
	}
	return exists, nil
}

func scanCurrency(row rowScanner) (domain.CurrencyInfo, error) {
	var info domain.CurrencyInfo

	err := row.Scan(
		&info.Code,
		&info.Name,
		&info.Ledger,
		&info.AssetScale,
	)
	if err != nil {
		return domain.CurrencyInfo{}, err
	}
	return info, nil
}

func sameCurrency(existing domain.CurrencyInfo, currency domain.CurrencyInfo) bool {
	return existing.Code == currency.Code &&
		existing.Name == currency.Name &&
		existing.Ledger == currency.Ledger &&
		existing.AssetScale == currency.AssetScale
}

func currencyError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("inserting currency info: %w", err)
	}
	if pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", currencies.ErrCurrencyConflict, pgErr.Detail)
	}
	return fmt.Errorf("inserting currency info: %w", err)
}
