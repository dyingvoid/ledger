package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
)

func toStatus(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case errors.Is(err, idempotency.ErrPreviousAttemptFailed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, idempotency.ErrInProgress):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, domain.ErrDepositAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, idempotency.ErrKeyRequired),
		errors.Is(err, idempotency.ErrKeyTooLong),
		errors.Is(err, domain.ErrAccountNameRequired),
		errors.Is(err, domain.ErrAccountUserRequired),
		errors.Is(err, domain.ErrAccountCurrencyRequired),
		errors.Is(err, domain.ErrAccountTypeInvalid),
		errors.Is(err, domain.ErrDepositAccountRequired),
		errors.Is(err, domain.ErrDepositAmountInvalid),
		errors.Is(err, domain.ErrDepositCurrencyRequired),
		errors.Is(err, domain.ErrDepositAccountNotClient),
		errors.Is(err, domain.ErrDepositCurrencyMismatch),
		errors.Is(err, accounts.ErrAccountTypeNotFound),
		errors.Is(err, accounts.ErrCurrencyNotFound):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
