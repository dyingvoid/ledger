package temporal

import (
	"context"
	"fmt"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountGateway interface {
	CreateAccountInfo(
		ctx context.Context,
		accountingID string,
		account domain.CreateAccountParams,
	) error
	OpenLedgerAccount(ctx context.Context, accountingID string) error
}

type Activities struct {
	accounts AccountGateway
}

func NewActivities(accounts AccountGateway) *Activities {
	return &Activities{accounts: accounts}
}

func (a *Activities) CreateAccountInfo(
	ctx context.Context,
	input domain.AccountOpeningInput,
) error {
	err := a.accounts.CreateAccountInfo(ctx, input.AccountingID, domain.CreateAccountParams{
		UserID:      input.UserID,
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
		Currency:    input.Currency,
	})
	if err != nil {
		return activityError("creating account info", err)
	}
	return nil
}

func (a *Activities) OpenLedgerAccount(ctx context.Context, accountingID string) error {
	if err := a.accounts.OpenLedgerAccount(ctx, accountingID); err != nil {
		return activityError("opening ledger account", err)
	}
	return nil
}

func activityError(message string, err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.FailedPrecondition:
		return temporal.NewNonRetryableApplicationError(message, "ACCOUNTING_REJECTED", err)
	case codes.Canceled,
		codes.Unknown,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss:
		return fmt.Errorf("%s: %w", message, err)
	case codes.OK:
		return fmt.Errorf("%s: unexpected gRPC OK status", message)
	}
	return fmt.Errorf("%s: unknown gRPC status", message)
}
