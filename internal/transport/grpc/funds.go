package grpc

import (
	"context"

	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
	fundsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/funds/v1"
)

type fundsService struct {
	fundsv1.UnimplementedFundsServiceServer
	deposits           *deposits.Service
	idempotencyService *idempotency.Service
}

func newFundsService(
	depositsService *deposits.Service,
	idempotencyService *idempotency.Service,
) *fundsService {
	return &fundsService{
		deposits:           depositsService,
		idempotencyService: idempotencyService,
	}
}

func (s *fundsService) Deposit(
	ctx context.Context,
	req *fundsv1.DepositRequest,
) (*fundsv1.DepositResponse, error) {
	key, err := idempotencyKey(ctx, "funds.deposit", req.GetAccountId())
	if err != nil {
		return nil, err
	}

	deposit, replayed, err := s.idempotencyService.Execute(ctx, key, func(ctx context.Context) (domain.Deposit, error) {
		return s.deposits.AddDeposit(ctx, deposits.AddDepositParams{
			AccountID:   req.GetAccountId(),
			AmountMinor: req.GetAmountMinor(),
			Currency:    domain.Currency(req.GetCurrency()),
		})
	})
	if err != nil {
		return nil, toStatus(err)
	}

	if err := setReplayHeader(ctx, replayed); err != nil {
		return nil, err
	}

	return &fundsv1.DepositResponse{
		JournalEntryId: deposit.JournalEntryID,
		Replayed:       replayed,
	}, nil
}
