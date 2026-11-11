package accounts

import (
	"context"

	"github.com/dyingvoid/ledger/internal/domain"
)

type InfoRepository interface {
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
	Get(ctx context.Context, id domain.ID) (domain.Account, error)
}

type LedgerRepository interface {
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
}

type Service struct {
	infos  InfoRepository
	ledger LedgerRepository
}

func NewService(infos InfoRepository, ledger LedgerRepository) *Service {
	return &Service{infos: infos, ledger: ledger}
}
