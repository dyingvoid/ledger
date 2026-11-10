package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
	accountsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/accounts/v1"
)

type accountService struct {
	accountsv1.UnimplementedAccountServiceServer
	accounts           *accounts.Service
	idempotencyService *idempotency.Service
}

func newAccountService(
	accountsService *accounts.Service,
	idempotencyService *idempotency.Service,
) *accountService {
	return &accountService{
		accounts:           accountsService,
		idempotencyService: idempotencyService,
	}
}

func (s *accountService) CreateAccount(
	ctx context.Context,
	req *accountsv1.CreateAccountRequest,
) (*accountsv1.CreateAccountResponse, error) {
	key, err := idempotencyKey(ctx, "accounts.create_account", req.GetUserId())
	if err != nil {
		return nil, err
	}

	account, replayed, err := s.idempotencyService.Execute(ctx, key, func(ctx context.Context) (domain.Account, error) {
		return s.accounts.CreateAccount(ctx, accounts.CreateAccountParams{
			UserID:      req.GetUserId(),
			Name:        req.GetName(),
			Description: req.GetDescription(),
			Type:        domain.AccountType(req.GetType()),
			Currency:    domain.Currency(req.GetCurrency()),
		})
	})
	if err != nil {
		return nil, toStatus(err)
	}

	if err := setReplayHeader(ctx, replayed); err != nil {
		return nil, err
	}

	return &accountsv1.CreateAccountResponse{
		Account:  toProtoAccount(account),
		Replayed: replayed,
	}, nil
}

func toProtoAccount(account domain.Account) *accountsv1.Account {
	response := &accountsv1.Account{
		Id:          account.ID,
		UserId:      account.UserID,
		Name:        account.Name,
		Description: account.Description,
		Type:        int32(account.Type),
		Currency:    string(account.Currency),
		DebitMinor:  account.DebitMinor,
		CreditMinor: account.CreditMinor,
	}
	if !account.CreatedAt.IsZero() {
		response.CreatedAt = timestamppb.New(account.CreatedAt)
	}
	return response
}
