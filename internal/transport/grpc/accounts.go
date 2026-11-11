package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
	accountsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/accounts/v1"
)

type accountService struct {
	accountsv1.UnimplementedAccountServiceServer
	accounts *accounts.Service
}

func newAccountService(accountsService *accounts.Service) *accountService {
	return &accountService{accounts: accountsService}
}

func (s *accountService) CreateAccountInfo(
	ctx context.Context,
	req *accountsv1.CreateAccountInfoRequest,
) (*accountsv1.CreateAccountInfoResponse, error) {
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}

	account, err := s.accounts.CreateAccountInfo(ctx, accounts.CreateAccountInfoParams{
		ID:          id,
		UserID:      req.GetUserId(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Type:        domain.AccountType(req.GetType()),
		Currency:    domain.Currency(req.GetCurrency()),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &accountsv1.CreateAccountInfoResponse{
		Account: toProtoAccount(account),
	}, nil
}

func (s *accountService) OpenLedgerAccount(
	ctx context.Context,
	req *accountsv1.OpenLedgerAccountRequest,
) (*accountsv1.OpenLedgerAccountResponse, error) {
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}

	account, err := s.accounts.OpenLedgerAccount(ctx, id)
	if err != nil {
		return nil, toStatus(err)
	}

	return &accountsv1.OpenLedgerAccountResponse{
		Account: toProtoAccount(account),
	}, nil
}

func toProtoAccount(account domain.Account) *accountsv1.AccountInfo {
	response := &accountsv1.AccountInfo{
		Id:          account.ID.String(),
		UserId:      account.UserID,
		Name:        account.Name,
		Description: account.Description,
		Type:        int32(account.Type),
		Currency:    string(account.Currency),
	}
	if !account.CreatedAt.IsZero() {
		response.CreatedAt = timestamppb.New(account.CreatedAt)
	}
	return response
}
