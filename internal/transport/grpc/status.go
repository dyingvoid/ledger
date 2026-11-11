package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/deposits"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

func toStatus(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case errors.Is(err, accounts.ErrAccountNotFound),
		errors.Is(err, domain_errors.ErrDepositAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, accounts.ErrAccountConflict),
		errors.Is(err, domain_errors.ErrDepositConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, accounts.ErrAccountStoreUnavailable),
		errors.Is(err, deposits.ErrLedgerUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, domain_errors.ErrAccountIDRequired),
		errors.Is(err, domain_errors.ErrAccountNameRequired),
		errors.Is(err, domain_errors.ErrAccountUserRequired),
		errors.Is(err, domain_errors.ErrAccountCurrencyRequired),
		errors.Is(err, domain_errors.ErrAccountTypeInvalid),
		errors.Is(err, domain_errors.ErrDepositIDRequired),
		errors.Is(err, domain_errors.ErrDepositAccountRequired),
		errors.Is(err, domain_errors.ErrDepositAmountInvalid),
		errors.Is(err, domain_errors.ErrDepositCurrencyRequired),
		errors.Is(err, domain_errors.ErrDepositAccountNotClient),
		errors.Is(err, domain_errors.ErrDepositCurrencyMismatch),
		errors.Is(err, accounts.ErrAccountTypeNotFound),
		errors.Is(err, accounts.ErrCurrencyNotFound),
		errors.Is(err, domain_errors.ErrIDInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
