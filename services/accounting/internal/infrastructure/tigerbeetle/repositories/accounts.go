package repositories

import (
	"context"
	"errors"
	"fmt"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/services/accounting/internal/core/accounts"
	"github.com/dyingvoid/ledger/services/accounting/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/services/accounting/internal/domain/errors"
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

func (r *Accounts) Create(
	ctx context.Context,
	account domain.AccountInfo,
) (domain.AccountLedger, error) {
	if account.ID.IsZero() {
		return domain.AccountLedger{}, domain_errors.ErrAccountIDRequired
	}

	info, err := r.currencies.Get(ctx, account.Currency)
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("getting currency ledger: %w", err)
	}

	flags, err := accountFlags(account.Type)
	if err != nil {
		return domain.AccountLedger{}, err
	}

	results, err := r.client.CreateAccounts([]tigerbeetle_go.Account{{
		ID:         toTigerbeetleID(account.ID),
		UserData64: uint64(account.UserID), //nolint:gosec // UserID is validated as positive before reaching the repository.
		Ledger:     info.Ledger,
		Code:       uint16(account.Type), //nolint:gosec // Account types are validated to be between 1000 and 5999.
		Flags:      flags,
	}})
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("%w: %w", accounts.ErrAccountStoreUnavailable, err)
	}
	if len(results) != 1 {
		return domain.AccountLedger{}, fmt.Errorf(
			"%w: expected 1 create result, got %d",
			accounts.ErrAccountStoreUnavailable,
			len(results),
		)
	}

	//nolint:exhaustive // Unknown statuses fall through to the generic error.
	switch results[0].Status {
	case tigerbeetle_go.AccountCreated, tigerbeetle_go.AccountExists:
		return newAccountLedger(account, info.Ledger)
	case tigerbeetle_go.AccountExistsWithDifferentUserData128,
		tigerbeetle_go.AccountExistsWithDifferentUserData64,
		tigerbeetle_go.AccountExistsWithDifferentUserData32:
		return domain.AccountLedger{}, r.ownershipConflict(ctx, account, results[0].Status)
	case tigerbeetle_go.AccountExistsWithDifferentFlags,
		tigerbeetle_go.AccountExistsWithDifferentLedger,
		tigerbeetle_go.AccountExistsWithDifferentCode:
		return domain.AccountLedger{}, fmt.Errorf("%w: %s", accounts.ErrAccountConflict, results[0].Status)
	default:
		return domain.AccountLedger{}, fmt.Errorf(
			"creating tigerbeetle account %s: %s",
			account.ID,
			results[0].Status,
		)
	}
}

func (r *Accounts) ownershipConflict(
	ctx context.Context,
	account domain.AccountInfo,
	status tigerbeetle_go.CreateAccountStatus,
) error {
	existing, err := r.Get(ctx, account.ID)
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return fmt.Errorf("%w: %s", accounts.ErrAccountConflict, account.ID)
	}
	if err != nil {
		return err
	}

	return fmt.Errorf(
		"%w: account %s exists with different user data (%s): stored user %d, requested user %d",
		domain_errors.ErrAccountOwnedByAnotherUser,
		account.ID,
		status,
		existing.UserID,
		account.UserID,
	)
}

func (r *Accounts) Get(_ context.Context, id domain.ID) (domain.AccountLedger, error) {
	found, err := r.client.LookupAccounts([]tigerbeetle_go.Uint128{toTigerbeetleID(id)})
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("%w: %w", accounts.ErrAccountStoreUnavailable, err)
	}
	if len(found) == 0 {
		return domain.AccountLedger{}, fmt.Errorf("%w: %s", accounts.ErrAccountNotFound, id)
	}
	if len(found) != 1 {
		return domain.AccountLedger{}, fmt.Errorf(
			"%w: expected 1 account, got %d",
			accounts.ErrAccountStoreUnavailable,
			len(found),
		)
	}

	account, err := domain.NewAccountLedger(
		fromTigerbeetleID(found[0].ID),
		int64(found[0].UserData64), //nolint:gosec // UserData64 stores a validated positive UserID.
		found[0].Ledger,
		domain.AccountType(found[0].Code),
	)
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("mapping ledger account: %w", err)
	}
	return account, nil
}

func newAccountLedger(
	account domain.AccountInfo,
	ledger uint32,
) (domain.AccountLedger, error) {
	opened, err := domain.NewAccountLedger(account.ID, account.UserID, ledger, account.Type)
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("mapping ledger account: %w", err)
	}
	return opened, nil
}

func accountFlags(accountType domain.AccountType) (uint16, error) {
	category, err := accountType.Category()
	if err != nil {
		return 0, fmt.Errorf("determining account category: %w", err)
	}

	flags := tigerbeetle_go.AccountFlags{}
	switch category {
	case domain.AccountCategoryAsset, domain.AccountCategoryExpense:
		flags.CreditsMustNotExceedDebits = true
	case domain.AccountCategoryLiability, domain.AccountCategoryEquity, domain.AccountCategoryRevenue:
		flags.DebitsMustNotExceedCredits = true
	default:
		return 0, fmt.Errorf("invalid account category: %d", category)
	}

	return flags.ToUint16(), nil
}
