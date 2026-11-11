package repositories

import (
	"context"
	"fmt"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const depositTransferCode uint16 = 1

type Deposits struct {
	client tigerbeetle_go.Client
}

func NewDeposits(client tigerbeetle_go.Client) *Deposits {
	return &Deposits{client: client}
}

func (r *Deposits) Create(_ context.Context, deposit domain.Deposit) (domain.Deposit, error) {
	return r.submitTransfer(deposit, buildDepositTransfer(deposit))
}

func buildDepositTransfer(deposit domain.Deposit) tigerbeetle_go.Transfer {
	clientAccountID := toTigerbeetleID(deposit.ClientAccount.ID)
	return tigerbeetle_go.Transfer{
		ID:              toTigerbeetleID(deposit.ID),
		DebitAccountID:  toTigerbeetleID(deposit.BankAccount.ID),
		CreditAccountID: clientAccountID,
		Amount:          tigerbeetle_go.ToUint128(uint64(deposit.AmountMinor)), //nolint:gosec // AmountMinor is validated as positive.
		UserData128:     clientAccountID,
		Ledger:          deposit.Currency.Ledger,
		Code:            depositTransferCode,
	}
}

func (r *Deposits) submitTransfer(
	deposit domain.Deposit,
	transfer tigerbeetle_go.Transfer,
) (domain.Deposit, error) {
	results, err := r.client.CreateTransfers([]tigerbeetle_go.Transfer{transfer})
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("%w: %w", deposits.ErrLedgerUnavailable, err)
	}
	if len(results) != 1 {
		return domain.Deposit{}, fmt.Errorf(
			"%w: expected 1 create result, got %d",
			deposits.ErrLedgerUnavailable,
			len(results),
		)
	}

	//nolint:exhaustive // Unknown statuses fall through to the unavailable error.
	switch results[0].Status {
	case tigerbeetle_go.TransferCreated:
		return deposit, nil
	case tigerbeetle_go.TransferExists:
		if err := r.verifyExistingTransfer(transfer); err != nil {
			return domain.Deposit{}, err
		}
		return deposit, nil
	case tigerbeetle_go.TransferDebitAccountNotFound:
		return domain.Deposit{}, fmt.Errorf(
			"%w: %s",
			domain_errors.ErrDepositBankAccountNotFound,
			deposit.BankAccount.ID,
		)
	case tigerbeetle_go.TransferCreditAccountNotFound:
		return domain.Deposit{}, fmt.Errorf(
			"%w: %s",
			domain_errors.ErrDepositClientAccountNotFound,
			deposit.ClientAccount.ID,
		)
	case tigerbeetle_go.TransferAccountsMustHaveTheSameLedger,
		tigerbeetle_go.TransferTransferMustHaveTheSameLedgerAsAccounts:
		return domain.Deposit{}, fmt.Errorf("%w: %s", domain_errors.ErrDepositCurrencyMismatch, results[0].Status)
	case tigerbeetle_go.TransferExistsWithDifferentFlags,
		tigerbeetle_go.TransferExistsWithDifferentPendingID,
		tigerbeetle_go.TransferExistsWithDifferentTimeout,
		tigerbeetle_go.TransferExistsWithDifferentDebitAccountID,
		tigerbeetle_go.TransferExistsWithDifferentCreditAccountID,
		tigerbeetle_go.TransferExistsWithDifferentAmount,
		tigerbeetle_go.TransferExistsWithDifferentUserData128,
		tigerbeetle_go.TransferExistsWithDifferentUserData64,
		tigerbeetle_go.TransferExistsWithDifferentUserData32,
		tigerbeetle_go.TransferExistsWithDifferentLedger,
		tigerbeetle_go.TransferExistsWithDifferentCode:
		return domain.Deposit{}, fmt.Errorf("%w: %s", domain_errors.ErrDepositConflict, results[0].Status)
	default:
		return domain.Deposit{}, fmt.Errorf(
			"%w: creating transfer %s: %s",
			deposits.ErrLedgerUnavailable,
			deposit.ID,
			results[0].Status,
		)
	}
}

func (r *Deposits) verifyExistingTransfer(expected tigerbeetle_go.Transfer) error {
	existing, err := r.client.LookupTransfers([]tigerbeetle_go.Uint128{expected.ID})
	if err != nil {
		return fmt.Errorf("%w: %w", deposits.ErrLedgerUnavailable, err)
	}
	if len(existing) != 1 {
		return fmt.Errorf("%w: existing transfer %s not found", domain_errors.ErrDepositConflict, expected.ID)
	}
	if !sameTransfer(existing[0], expected) {
		return fmt.Errorf("%w: transfer %s", domain_errors.ErrDepositConflict, expected.ID)
	}
	return nil
}

func sameTransfer(actual tigerbeetle_go.Transfer, expected tigerbeetle_go.Transfer) bool {
	return actual.ID == expected.ID &&
		actual.DebitAccountID == expected.DebitAccountID &&
		actual.CreditAccountID == expected.CreditAccountID &&
		actual.Amount == expected.Amount &&
		actual.UserData128 == expected.UserData128 &&
		actual.Ledger == expected.Ledger &&
		actual.Code == expected.Code &&
		actual.Flags == expected.Flags
}
