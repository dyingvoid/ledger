package accounts

import (
	"context"

	"github.com/dyingvoid/ledger/internal/domain"
)

type AccountRepository interface {
	AccountTypeExists(ctx context.Context, accountType domain.AccountType) (bool, error)
	CurrencyExists(ctx context.Context, currency domain.Currency) (bool, error)
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
}

type Service struct {
	accounts AccountRepository
}

func NewService(accounts AccountRepository) *Service {
	return &Service{accounts: accounts}
}
