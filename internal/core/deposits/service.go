package deposits

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

var (
	ErrLedgerUnavailable = errors.New("ledger unavailable")
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

type DepositParams struct {
	ID          domain.ID
	AccountID   domain.ID
	AmountMinor int64
	Currency    domain.Currency
}

// Deposit moves funds into a client account. The deposit ID is the
// TigerBeetle transfer ID and therefore its idempotency key.
func (s *Service) Deposit(ctx context.Context, params DepositParams) (domain.Deposit, error) {
	deposit, err := domain.NewDeposit(
		params.ID,
		params.AccountID,
		params.AmountMinor,
		params.Currency,
	)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("validating deposit: %w", err)
	}

	created, err := s.deposits.Create(ctx, deposit)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("creating deposit: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"deposit created",
		"transfer_id", created.ID,
		"account_id", created.AccountID,
		"amount_minor", created.AmountMinor,
		"currency", created.Currency,
	)

	return created, nil
}
