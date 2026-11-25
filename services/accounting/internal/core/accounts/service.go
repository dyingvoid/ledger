package accounts

import (
	"context"

	"github.com/dyingvoid/ledger/services/accounting/internal/domain"
)

type InfoRepository interface {
	Create(ctx context.Context, account domain.AccountInfo) (domain.AccountInfo, error)
	Get(ctx context.Context, id domain.ID) (domain.AccountInfo, error)
}

type LedgerRepository interface {
	Create(ctx context.Context, account domain.AccountInfo) (domain.AccountLedger, error)
	Get(ctx context.Context, id domain.ID) (domain.AccountLedger, error)
}

type Service struct {
	infos  InfoRepository
	ledger LedgerRepository
}

func NewService(infos InfoRepository, ledger LedgerRepository) *Service {
	return &Service{infos: infos, ledger: ledger}
}
