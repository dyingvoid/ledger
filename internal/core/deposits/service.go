package deposits

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

var (
	ErrDepositStoreUnavailable = errors.New("deposit store unavailable")
	ErrDepositConflict         = errors.New("deposit conflicts with an existing transfer")
	ErrTransferIDRequired      = errors.New("deposit transfer ID is required")
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
	AccountID         int64
	ExternalAccountID domain.ExternalID
	TransferID        domain.ExternalID
	AmountMinor       int64
	Currency          domain.Currency
}

func (s *Service) AddDeposit(ctx context.Context, params AddDepositParams) (domain.Deposit, error) {
	deposit, err := domain.NewDeposit(
		params.AccountID,
		params.ExternalAccountID,
		params.AmountMinor,
		params.Currency,
	)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("validating deposit: %w", err)
	}

	deposit.TransferID = params.TransferID

	created, err := s.deposits.Create(ctx, deposit)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("creating deposit: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"deposit added",
		"account_id", created.AccountID,
		"external_account_id", created.ExternalAccountID.String(),
		"journal_entry_id", created.JournalEntryID,
		"transfer_id", created.TransferID.String(),
		"amount_minor", created.AmountMinor,
		"currency", created.Currency,
	)

	return created, nil
}
