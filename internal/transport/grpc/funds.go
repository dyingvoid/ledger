package grpc

import (
	"context"

	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/domain"
	fundsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/funds/v1"
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
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}

	accountID, err := parseID(req.GetAccountId())
	if err != nil {
		return nil, err
	}

	deposit, err := s.deposits.Deposit(ctx, deposits.DepositParams{
		ID:          id,
		AccountID:   accountID,
		AmountMinor: req.GetAmountMinor(),
		Currency:    domain.Currency(req.GetCurrency()),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &fundsv1.DepositResponse{
		TransferId: deposit.ID.String(),
	}, nil
}
