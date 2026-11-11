package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
)

const (
	accountTypeForeignKey     = "account_info_account_type_fkey"
	accountCurrencyForeignKey = "account_info_currency_fkey"
)

type Accounts struct {
	pool *pgxpool.Pool
}

func NewAccounts(pool *pgxpool.Pool) *Accounts {
	return &Accounts{pool: pool}
}

func (r *Accounts) Create(ctx context.Context, account domain.Account) (domain.Account, error) {
	id := account.ID.Bytes()

	created, err := scanAccount(r.pool.QueryRow(
		ctx,
		`INSERT INTO ledger.account_info (id, user_id, name, description, account_type, currency)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO NOTHING
		 RETURNING id, user_id, name, description, account_type, currency, deleted_at, created_at`,
		id[:],
		account.UserID,
		account.Name,
		account.Description,
		int32(account.Type),
		account.Currency,
	))
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, accountError(err)
	}

	existing, err := r.Get(ctx, account.ID)
	if err != nil {
		return domain.Account{}, err
	}
	if !sameAccountInfo(existing, account) {
		return domain.Account{}, fmt.Errorf("%w: %s", accounts.ErrAccountConflict, account.ID)
	}

	return existing, nil
}

func (r *Accounts) Get(ctx context.Context, id domain.ID) (domain.Account, error) {
	stored := id.Bytes()

	account, err := scanAccount(r.pool.QueryRow(
		ctx,
		`SELECT id, user_id, name, description, account_type, currency, deleted_at, created_at
		 FROM ledger.account_info
		 WHERE id = $1
		   AND deleted_at IS NULL`,
		stored[:],
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, fmt.Errorf("%w: %s", accounts.ErrAccountNotFound, id)
	}
	if err != nil {
		return domain.Account{}, fmt.Errorf("getting account info: %w", err)
	}

	return account, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row rowScanner) (domain.Account, error) {
	var (
		account  domain.Account
		stored   []byte
		typ      int32
		currency string
	)

	err := row.Scan(
		&stored,
		&account.UserID,
		&account.Name,
		&account.Description,
		&typ,
		&currency,
		&account.DeletedAt,
		&account.CreatedAt,
	)
	if err != nil {
		return domain.Account{}, err
	}
	if len(stored) != 16 {
		return domain.Account{}, fmt.Errorf("account info ID has %d bytes, want 16", len(stored))
	}

	account.ID = domain.IDFromBytes([16]byte(stored))
	account.Type = domain.AccountType(typ)
	account.Currency = currency
	return account, nil
}

func sameAccountInfo(existing domain.Account, account domain.Account) bool {
	return existing.ID == account.ID &&
		existing.UserID == account.UserID &&
		existing.Name == account.Name &&
		existing.Description == account.Description &&
		existing.Type == account.Type &&
		existing.Currency == account.Currency
}

func accountError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("inserting account info: %w", err)
	}

	switch pgErr.ConstraintName {
	case accountTypeForeignKey:
		return fmt.Errorf("%w: %s", accounts.ErrAccountTypeNotFound, pgErr.Detail)
	case accountCurrencyForeignKey:
		return fmt.Errorf("%w: %s", accounts.ErrCurrencyNotFound, pgErr.Detail)
	default:
		return fmt.Errorf("inserting account info: %w", err)
	}
}
