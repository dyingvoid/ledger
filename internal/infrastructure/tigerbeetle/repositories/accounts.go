package repositories

import (
	"context"
	"fmt"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

type CurrencyLedgerRepository interface {
	Get(ctx context.Context, currency string) (domain.CurrencyInfo, error)
}

type Accounts struct {
	client     tigerbeetle_go.Client
	currencies CurrencyLedgerRepository
}

func NewAccounts(client tigerbeetle_go.Client, currencies CurrencyLedgerRepository) *Accounts {
	return &Accounts{client: client, currencies: currencies}
}

func (r *Accounts) Create(ctx context.Context, account domain.Account) (domain.Account, error) {
	if account.ID.IsZero() {
		return domain.Account{}, domain_errors.ErrAccountIDRequired
	}

	info, err := r.currencies.Get(ctx, account.Currency)
	if err != nil {
		return domain.Account{}, fmt.Errorf("getting currency ledger: %w", err)
	}

	flags, err := accountFlags(account)
	if err != nil {
		return domain.Account{}, err
	}

	results, err := r.client.CreateAccounts([]tigerbeetle_go.Account{{
		ID:         toTigerbeetleID(account.ID),
		UserData64: uint64(account.UserID), //nolint:gosec // UserID is validated as positive before reaching the repository.
		Ledger:     info.Ledger,
		Code:       uint16(account.Type), //nolint:gosec // Account types are validated to be between 1000 and 5999.
		Flags:      flags,
	}})
	if err != nil {
		return domain.Account{}, fmt.Errorf("%w: %w", accounts.ErrAccountStoreUnavailable, err)
	}
	if len(results) != 1 {
		return domain.Account{}, fmt.Errorf(
			"%w: expected 1 create result, got %d",
			accounts.ErrAccountStoreUnavailable,
			len(results),
		)
	}

	//nolint:exhaustive // Unknown statuses fall through to the generic error.
	switch results[0].Status {
	case tigerbeetle_go.AccountCreated, tigerbeetle_go.AccountExists:
		return account, nil
	case tigerbeetle_go.AccountExistsWithDifferentFlags,
		tigerbeetle_go.AccountExistsWithDifferentUserData128,
		tigerbeetle_go.AccountExistsWithDifferentUserData64,
		tigerbeetle_go.AccountExistsWithDifferentUserData32,
		tigerbeetle_go.AccountExistsWithDifferentLedger,
		tigerbeetle_go.AccountExistsWithDifferentCode:
		return domain.Account{}, fmt.Errorf("%w: %s", accounts.ErrAccountConflict, results[0].Status)
	default:
		return domain.Account{}, fmt.Errorf(
			"creating tigerbeetle account %s: %s",
			account.ID,
			results[0].Status,
		)
	}
}

func accountFlags(account domain.Account) (uint16, error) {
	side, err := account.NormalBalance()
	if err != nil {
		return 0, fmt.Errorf("determining account normal balance: %w", err)
	}

	flags := tigerbeetle_go.AccountFlags{}
	switch side {
	case domain.Debit:
		flags.CreditsMustNotExceedDebits = true
	case domain.Credit:
		flags.DebitsMustNotExceedCredits = true
	default:
		return 0, fmt.Errorf("invalid account balance side: %d", side)
	}

	return flags.ToUint16(), nil
}
