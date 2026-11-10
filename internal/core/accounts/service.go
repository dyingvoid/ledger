package accounts

import (
	"context"

	"github.com/dyingvoid/ledger/internal/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
}

type AccountTypeRepository interface {
	AccountTypeExists(ctx context.Context, accountType domain.AccountType) (bool, error)
}

type CurrencyRepository interface {
	Exists(ctx context.Context, currency domain.Currency) (bool, error)
}

type Service struct {
	accounts     AccountRepository
	accountTypes AccountTypeRepository
	currencies   CurrencyRepository
}

func NewService(
	accounts AccountRepository,
	accountTypes AccountTypeRepository,
	currencies CurrencyRepository,
) *Service {
	return &Service{
		accounts:     accounts,
		accountTypes: accountTypes,
		currencies:   currencies,
	}
}
