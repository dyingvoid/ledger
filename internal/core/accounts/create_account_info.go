package accounts

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

var (
	ErrAccountNotFound         = errors.New("account not found")
	ErrAccountConflict         = errors.New("account already exists with different data")
	ErrAccountStoreUnavailable = errors.New("account store unavailable")
	ErrAccountTypeNotFound     = errors.New("account type not found")
	ErrCurrencyNotFound        = errors.New("currency not found")
)

type CreateAccountInfoParams struct {
	ID          domain.ID
	UserID      int64
	Name        string
	Description string
	Type        domain.AccountType
	Currency    domain.Currency
}

func (s *Service) CreateAccountInfo(
	ctx context.Context,
	params CreateAccountInfoParams,
) (domain.Account, error) {
	account, err := domain.NewAccount(
		params.ID,
		params.UserID,
		params.Name,
		params.Description,
		params.Type,
		params.Currency,
	)
	if err != nil {
		return domain.Account{}, fmt.Errorf("validating account: %w", err)
	}

	created, err := s.infos.Create(ctx, account)
	if err != nil {
		return domain.Account{}, fmt.Errorf("creating account info: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"account info created",
		"account_id", created.ID,
		"user_id", created.UserID,
	)

	return created, nil
}
