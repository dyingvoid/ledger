package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
)

type Accounts struct {
	defaultExecutor postgres.Executor
}

func NewAccounts(pool *pgxpool.Pool) *Accounts {
	return &Accounts{defaultExecutor: pool}
}

func (r *Accounts) AccountTypeExists(ctx context.Context, accountType domain.AccountType) (bool, error) {
	var exists bool
	err := r.executor(ctx).QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM ledger.account_type WHERE code = $1)`,
		int32(accountType),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking account type: %w", err)
	}
	return exists, nil
}

func (r *Accounts) CurrencyExists(ctx context.Context, currency domain.Currency) (bool, error) {
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

func (r *Accounts) Create(ctx context.Context, account domain.Account) (domain.Account, error) {
	err := r.executor(ctx).QueryRow(
		ctx,
		`INSERT INTO ledger.account (user_id, name, description, account_type, currency)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, dr, cr, created_at`,
		account.UserID,
		account.Name,
		account.Description,
		int32(account.Type),
		string(account.Currency),
	).Scan(
		&account.ID,
		&account.DebitMinor,
		&account.CreditMinor,
		&account.CreatedAt,
	)
	if err != nil {
		return domain.Account{}, fmt.Errorf("inserting account: %w", err)
	}

	return account, nil
}

func (r *Accounts) executor(ctx context.Context) postgres.Executor {
	return postgres.ExecutorFrom(ctx, r.defaultExecutor)
}
