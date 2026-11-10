package deposits

import (
	"context"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

type Repository interface {
	Create(ctx context.Context, deposit domain.Deposit) (domain.Deposit, error)
}

type Service struct {
	deposits Repository
}

func NewService(deposits Repository) *Service {
	return &Service{deposits: deposits}
}

type AddDepositParams struct {
	AccountID   int64
	AmountMinor int64
	Currency    domain.Currency
}

func (s *Service) AddDeposit(ctx context.Context, params AddDepositParams) (domain.Deposit, error) {
	deposit, err := domain.NewDeposit(params.AccountID, params.AmountMinor, params.Currency)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("validating deposit: %w", err)
	}

	created, err := s.deposits.Create(ctx, deposit)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("creating deposit: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"deposit added",
		"account_id", created.AccountID,
		"journal_entry_id", created.JournalEntryID,
		"amount_minor", created.AmountMinor,
		"currency", created.Currency,
	)

	return created, nil
}
