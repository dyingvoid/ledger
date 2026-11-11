package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/currencies"
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
		errors.Is(err, accounts.ErrBankCashAccountNotFound),
		errors.Is(err, domain_errors.ErrDepositClientAccountNotFound),
		errors.Is(err, domain_errors.ErrDepositBankAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, accounts.ErrAccountConflict),
		errors.Is(err, currencies.ErrCurrencyConflict),
		errors.Is(err, domain_errors.ErrDepositConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, accounts.ErrAccountStoreUnavailable),
		errors.Is(err, deposits.ErrLedgerUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, domain_errors.ErrDepositAccountDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain_errors.ErrAccountIDRequired),
		errors.Is(err, domain_errors.ErrAccountNameRequired),
		errors.Is(err, domain_errors.ErrAccountUserRequired),
		errors.Is(err, domain_errors.ErrAccountCurrencyRequired),
		errors.Is(err, domain_errors.ErrAccountTypeInvalid),
		errors.Is(err, domain_errors.ErrDepositIDRequired),
		errors.Is(err, domain_errors.ErrDepositAmountInvalid),
		errors.Is(err, domain_errors.ErrDepositCurrencyRequired),
		errors.Is(err, domain_errors.ErrDepositClientAccountRequired),
		errors.Is(err, domain_errors.ErrDepositBankAccountRequired),
		errors.Is(err, domain_errors.ErrDepositAccountsIdentical),
		errors.Is(err, domain_errors.ErrDepositClientAccountTypeUnsupported),
		errors.Is(err, domain_errors.ErrDepositBankAccountTypeUnsupported),
		errors.Is(err, domain_errors.ErrDepositClientCurrencyMismatch),
		errors.Is(err, domain_errors.ErrDepositBankCurrencyMismatch),
		errors.Is(err, domain_errors.ErrDepositCurrencyMismatch),
		errors.Is(err, domain_errors.ErrCurrencyCodeRequired),
		errors.Is(err, domain_errors.ErrCurrencyNameRequired),
		errors.Is(err, domain_errors.ErrCurrencyLedgerInvalid),
		errors.Is(err, domain_errors.ErrCurrencyAssetScaleInvalid),
		errors.Is(err, accounts.ErrAccountTypeNotFound),
		errors.Is(err, accounts.ErrCurrencyNotFound),
		errors.Is(err, domain_errors.ErrIDInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
