package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const depositTransferCode uint16 = 1

type Deposits struct {
	client          tigerbeetle_go.Client
	currencies      CurrencyLedgerRepository
	bankCashIDCache sync.Map
}

func NewDeposits(client tigerbeetle_go.Client, currencies CurrencyLedgerRepository) *Deposits {
	return &Deposits{
		client:     client,
		currencies: currencies,
	}
}

func (r *Deposits) Create(ctx context.Context, deposit domain.Deposit) (domain.Deposit, error) {
	info, err := r.currencyInfo(ctx, deposit.Currency)
	if err != nil {
		return domain.Deposit{}, err
	}

	clientAccountID, err := r.clientAccountID(deposit, info.Ledger)
	if err != nil {
		return domain.Deposit{}, err
	}

	bankCashAccountID, err := r.ensureBankCashAccount(info.Ledger)
	if err != nil {
		return domain.Deposit{}, err
	}

	return r.submitTransfer(deposit, buildDepositTransfer(
		deposit,
		bankCashAccountID,
		clientAccountID,
		info.Ledger,
	))
}

func (r *Deposits) currencyInfo(ctx context.Context, currency domain.Currency) (domain.CurrencyInfo, error) {
	info, err := r.currencies.Get(ctx, currency)
	if err != nil {
		return domain.CurrencyInfo{}, fmt.Errorf("getting currency ledger: %w", err)
	}
	return info, nil
}

func (r *Deposits) clientAccountID(
	deposit domain.Deposit,
	ledger uint32,
) (tigerbeetle_go.Uint128, error) {
	accountID := toTigerbeetleID(deposit.AccountID)
	accountsFound, err := r.client.LookupAccounts([]tigerbeetle_go.Uint128{accountID})
	if err != nil {
		return tigerbeetle_go.Uint128{}, fmt.Errorf("%w: %w", deposits.ErrLedgerUnavailable, err)
	}
	if len(accountsFound) == 0 {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: %s",
			domain_errors.ErrDepositAccountNotFound,
			deposit.AccountID,
		)
	}

	account := accountsFound[0]
	if account.Code != uint16(domain.AccountTypeClientDeposit) {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: %s",
			domain_errors.ErrDepositAccountNotClient,
			deposit.AccountID,
		)
	}
	if account.Ledger != ledger {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: account ledger %d, deposit ledger %d",
			domain_errors.ErrDepositCurrencyMismatch,
			account.Ledger,
			ledger,
		)
	}

	return accountID, nil
}

func buildDepositTransfer(
	deposit domain.Deposit,
	bankCashAccountID tigerbeetle_go.Uint128,
	clientAccountID tigerbeetle_go.Uint128,
	ledger uint32,
) tigerbeetle_go.Transfer {
	return tigerbeetle_go.Transfer{
		ID:              toTigerbeetleID(deposit.ID),
		DebitAccountID:  bankCashAccountID,
		CreditAccountID: clientAccountID,
		Amount:          tigerbeetle_go.ToUint128(uint64(deposit.AmountMinor)), //nolint:gosec // AmountMinor is validated as positive.
		UserData128:     clientAccountID,
		Ledger:          ledger,
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
	case tigerbeetle_go.TransferDebitAccountNotFound, tigerbeetle_go.TransferCreditAccountNotFound:
		return domain.Deposit{}, fmt.Errorf(
			"%w: %s",
			domain_errors.ErrDepositAccountNotFound,
			deposit.AccountID,
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

func (r *Deposits) ensureBankCashAccount(ledger uint32) (tigerbeetle_go.Uint128, error) {
	if cached, ok := r.bankCashIDCache.Load(ledger); ok {
		accountID, ok := cached.(tigerbeetle_go.Uint128)
		if !ok {
			return tigerbeetle_go.Uint128{}, fmt.Errorf(
				"%w: invalid bank cash account cache entry",
				deposits.ErrLedgerUnavailable,
			)
		}
		return accountID, nil
	}

	externalID, err := deterministicID(
		"ledger.bank-cash.v1",
		strconv.FormatUint(uint64(ledger), 10),
	)
	if err != nil {
		return tigerbeetle_go.Uint128{}, fmt.Errorf("deriving bank cash account ID: %w", err)
	}

	accountID := toTigerbeetleID(externalID)
	flags := tigerbeetle_go.AccountFlags{CreditsMustNotExceedDebits: true}
	results, err := r.client.CreateAccounts([]tigerbeetle_go.Account{{
		ID:     accountID,
		Ledger: ledger,
		Code:   uint16(domain.AccountTypeBankCash),
		Flags:  flags.ToUint16(),
	}})
	if err != nil {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: creating bank cash account: %w",
			deposits.ErrLedgerUnavailable,
			err,
		)
	}
	if len(results) != 1 {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: expected 1 bank cash create result, got %d",
			deposits.ErrLedgerUnavailable,
			len(results),
		)
	}

	//nolint:exhaustive // Unknown statuses fall through to the unavailable error.
	switch results[0].Status {
	case tigerbeetle_go.AccountCreated, tigerbeetle_go.AccountExists:
		r.bankCashIDCache.Store(ledger, accountID)
		return accountID, nil
	case tigerbeetle_go.AccountExistsWithDifferentFlags,
		tigerbeetle_go.AccountExistsWithDifferentUserData128,
		tigerbeetle_go.AccountExistsWithDifferentUserData64,
		tigerbeetle_go.AccountExistsWithDifferentUserData32,
		tigerbeetle_go.AccountExistsWithDifferentLedger,
		tigerbeetle_go.AccountExistsWithDifferentCode:
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: bank cash account for ledger %d: %s",
			domain_errors.ErrDepositConflict,
			ledger,
			results[0].Status,
		)
	default:
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: creating bank cash account for ledger %d: %s",
			deposits.ErrLedgerUnavailable,
			ledger,
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

func deterministicID(namespace string, value string) (domain.ID, error) {
	digest := sha256.Sum256([]byte(namespace + ":" + value))
	return domain.ParseID("0x" + hex.EncodeToString(digest[:16]))
}
