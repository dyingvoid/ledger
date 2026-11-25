package grpc

import (
	"context"

	"github.com/dyingvoid/ledger/services/accounting/internal/core/deposits"
	"github.com/dyingvoid/ledger/services/accounting/internal/domain"
	fundsv1 "github.com/dyingvoid/ledger/services/accounting/pkg/proto/ledger/funds/v1"
)

type fundsService struct {
	fundsv1.UnimplementedFundsServiceServer
	deposits *deposits.Service
}

func newFundsService(depositsService *deposits.Service) *fundsService {
	return &fundsService{deposits: depositsService}
}

func (s *fundsService) Deposit(
	ctx context.Context,
	req *fundsv1.DepositRequest,
) (*fundsv1.DepositResponse, error) {
	id, err := domain.ParseID(req.GetId())
	if err != nil {
		return nil, err
	}

	accountID, err := domain.ParseID(req.GetAccountId())
	if err != nil {
		return nil, err
	}

	deposit, err := s.deposits.Deposit(ctx, deposits.DepositParams{
		ID:          id,
		AccountID:   accountID,
		AmountMinor: req.GetAmountMinor(),
		Currency:    req.GetCurrency(),
	})
	if err != nil {
		return nil, err
	}

	return &fundsv1.DepositResponse{
		TransferId: deposit.ID.String(),
	}, nil
}
