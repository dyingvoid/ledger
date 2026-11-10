package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
)

const (
	depositAmountInvalidCode    = "LEDG1"
	depositAccountNotClientCode = "LEDG2"
	depositCurrencyMismatchCode = "LEDG3"
)

type Deposits struct {
	defaultExecutor postgres.Executor
}

func NewDeposits(pool *pgxpool.Pool) *Deposits {
	return &Deposits{defaultExecutor: pool}
}

func (r *Deposits) Create(ctx context.Context, deposit domain.Deposit) (domain.Deposit, error) {
	err := r.executor(ctx).QueryRow(
		ctx,
		`SELECT journal_entry_id
		 FROM ledger.add_deposit($1, $2, $3)`,
		deposit.AccountID,
		deposit.AmountMinor,
		string(deposit.Currency),
	).Scan(&deposit.JournalEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deposit{}, fmt.Errorf("%w: %d", domain.ErrDepositAccountNotFound, deposit.AccountID)
	}
	if err != nil {
		return domain.Deposit{}, depositError(err)
	}

	return deposit, nil
}

func depositError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("adding deposit: %w", err)
	}

	switch pgErr.Code {
	case depositAmountInvalidCode:
		return fmt.Errorf("%w: %s", domain.ErrDepositAmountInvalid, pgErr.Message)
	case depositAccountNotClientCode:
		return fmt.Errorf("%w: %s", domain.ErrDepositAccountNotClient, pgErr.Message)
	case depositCurrencyMismatchCode:
		return fmt.Errorf("%w: %s", domain.ErrDepositCurrencyMismatch, pgErr.Message)
	default:
		return fmt.Errorf("adding deposit: %w", err)
	}
}

func (r *Deposits) executor(ctx context.Context) postgres.Executor {
	return postgres.ExecutorFrom(ctx, r.defaultExecutor)
}
