package accounts

import (
	"context"

	"github.com/dyingvoid/ledger/internal/domain"
)

type AccountRepository interface {
	AccountTypeExists(ctx context.Context, accountType domain.AccountType) (bool, error)
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
}

type CurrencyRepository interface {
	Exists(ctx context.Context, currency domain.Currency) (bool, error)
}

type Service struct {
	accounts   AccountRepository
	currencies CurrencyRepository
}

func NewService(accounts AccountRepository, currencies CurrencyRepository) *Service {
	return &Service{accounts: accounts, currencies: currencies}
}
