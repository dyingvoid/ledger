package accounts

import (
	"context"

	"github.com/dyingvoid/ledger/services/accounting/internal/domain"
)

// TODO: must work with domain AccountInfo
type InfoRepository interface {
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
	Get(ctx context.Context, id domain.ID) (domain.Account, error)
}

// TODO: introduce separate account domain entity for ledger data
type LedgerRepository interface {
	Create(ctx context.Context, account domain.Account) (domain.Account, error)
	Get(ctx context.Context, id domain.ID) (domain.Account, error)
}

type Service struct {
	infos  InfoRepository
	ledger LedgerRepository
}

func NewService(infos InfoRepository, ledger LedgerRepository) *Service {
	return &Service{infos: infos, ledger: ledger}
}
