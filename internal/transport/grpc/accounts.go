package grpc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
	accountsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/accounts/v1"
)

const idempotencyKeyHeader = "x-idempotency-key"

type accountService struct {
	accountsv1.UnimplementedAccountServiceServer
	accounts      *accounts.Service
	createAccount *idempotency.Runner[domain.Account]
}

func newAccountService(
	accountsService *accounts.Service,
	createAccount *idempotency.Runner[domain.Account],
) *accountService {
	return &accountService{
		accounts:      accountsService,
		createAccount: createAccount,
	}
}

func (s *accountService) CreateAccount(
	ctx context.Context,
	req *accountsv1.CreateAccountRequest,
) (*accountsv1.CreateAccountResponse, error) {
	key, err := idempotencyKey(ctx, req)
	if err != nil {
		return nil, err
	}
	
	account, replayed, err := s.createAccount.Do(ctx, key, func(ctx context.Context) (domain.Account, error) {
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

	if replayed {
		if err := grpcpkg.SetHeader(ctx, metadata.Pairs("x-idempotent-replayed", "true")); err != nil {
			return nil, status.Errorf(codes.Internal, "setting replay header: %v", err)
		}
	}

	return &accountsv1.CreateAccountResponse{
		Account:  toProtoAccount(account),
		Replayed: replayed,
	}, nil
}

func idempotencyKey(ctx context.Context, req *accountsv1.CreateAccountRequest) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "%s metadata is required", idempotencyKeyHeader)
	}

	values := md.Get(idempotencyKeyHeader)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return "", status.Errorf(codes.InvalidArgument, "%s metadata is required", idempotencyKeyHeader)
	}

	return "accounts.create_account:" + strconv.FormatInt(req.GetUserId(), 10) + ":" + strings.TrimSpace(values[0]), nil
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

func toStatus(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case errors.Is(err, idempotency.ErrPreviousAttemptFailed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, idempotency.ErrKeyRequired),
		errors.Is(err, idempotency.ErrKeyTooLong),
		errors.Is(err, domain.ErrAccountNameRequired),
		errors.Is(err, domain.ErrAccountUserRequired),
		errors.Is(err, domain.ErrAccountCurrencyRequired),
		errors.Is(err, domain.ErrAccountTypeInvalid),
		errors.Is(err, accounts.ErrAccountTypeNotFound),
		errors.Is(err, accounts.ErrCurrencyNotFound):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
