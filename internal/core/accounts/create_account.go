package accounts

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

var (
	ErrAccountTypeNotFound = errors.New("account type not found")
	ErrCurrencyNotFound    = errors.New("currency not found")
)

type CreateAccountParams struct {
	UserID      int64
	Name        string
	Description string
	Type        domain.AccountType
	Currency    domain.Currency
}

func (s *Service) CreateAccount(ctx context.Context, params CreateAccountParams) (domain.Account, error) {
	account, err := domain.NewAccount(
		params.UserID,
		params.Name,
		params.Description,
		params.Type,
		params.Currency,
	)
	if err != nil {
		return domain.Account{}, fmt.Errorf("validating account: %w", err)
	}

	if err := s.validateReferenceData(ctx, account); err != nil {
		return domain.Account{}, err
	}

	created, err := s.accounts.Create(ctx, account)
	if err != nil {
		return domain.Account{}, fmt.Errorf("creating account: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"account created",
		"account_id", created.ID,
		"user_id", created.UserID,
	)

	return created, nil
}

func (s *Service) validateReferenceData(ctx context.Context, account domain.Account) error {
	accountTypeExists, err := s.accounts.AccountTypeExists(ctx, account.Type)
	if err != nil {
		return fmt.Errorf("checking account type: %w", err)
	}
	if !accountTypeExists {
		return fmt.Errorf("%w: %d", ErrAccountTypeNotFound, account.Type)
	}

	currencyExists, err := s.currencies.Exists(ctx, account.Currency)
	if err != nil {
		return fmt.Errorf("checking currency: %w", err)
	}
	if !currencyExists {
		return fmt.Errorf("%w: %q", ErrCurrencyNotFound, account.Currency)
	}

	return nil
}
