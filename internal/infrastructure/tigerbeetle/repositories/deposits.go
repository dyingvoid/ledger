package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/domain"
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
	if err := validateDeposit(deposit); err != nil {
		return domain.Deposit{}, err
	}

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

func validateDeposit(deposit domain.Deposit) error {
	if deposit.ExternalAccountID.IsZero() {
		return fmt.Errorf("%w: external account ID is required", domain.ErrDepositAccountRequired)
	}
	if deposit.TransferID.IsZero() {
		return deposits.ErrTransferIDRequired
	}
	return nil
}

func (r *Deposits) currencyInfo(ctx context.Context, currency domain.Currency) (domain.CurrencyInfo, error) {
	info, err := r.currencies.Get(ctx, currency)
	if err != nil {
		if errors.Is(err, accounts.ErrCurrencyNotFound) {
			return domain.CurrencyInfo{}, fmt.Errorf("%w: %q", accounts.ErrCurrencyNotConfigured, currency)
		}
		return domain.CurrencyInfo{}, fmt.Errorf("getting currency ledger: %w", err)
	}
	return info, nil
}

func (r *Deposits) clientAccountID(
	deposit domain.Deposit,
	ledger uint32,
) (tigerbeetle_go.Uint128, error) {
	accountID := tigerbeetle_go.BytesToUint128(deposit.ExternalAccountID.Bytes())
	accountsFound, err := r.client.LookupAccounts([]tigerbeetle_go.Uint128{accountID})
	if err != nil {
		return tigerbeetle_go.Uint128{}, fmt.Errorf("%w: %w", deposits.ErrDepositStoreUnavailable, err)
	}
	if len(accountsFound) == 0 {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: %s",
			domain.ErrDepositAccountNotFound,
			deposit.ExternalAccountID,
		)
	}

	account := accountsFound[0]
	if account.Code != uint16(domain.AccountTypeClientDeposit) {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: %s",
			domain.ErrDepositAccountNotClient,
			deposit.ExternalAccountID,
		)
	}
	if account.Ledger != ledger {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: account ledger %d, deposit ledger %d",
			domain.ErrDepositCurrencyMismatch,
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
		ID:              tigerbeetle_go.BytesToUint128(deposit.TransferID.Bytes()),
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
		return domain.Deposit{}, fmt.Errorf("%w: %w", deposits.ErrDepositStoreUnavailable, err)
	}
	if len(results) != 1 {
		return domain.Deposit{}, fmt.Errorf(
			"%w: expected 1 create result, got %d",
			deposits.ErrDepositStoreUnavailable,
			len(results),
		)
	}

	//nolint:exhaustive // Unknown statuses fall through to the store-unavailable error.
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
			domain.ErrDepositAccountNotFound,
			deposit.ExternalAccountID,
		)
	case tigerbeetle_go.TransferAccountsMustHaveTheSameLedger,
		tigerbeetle_go.TransferTransferMustHaveTheSameLedgerAsAccounts:
		return domain.Deposit{}, fmt.Errorf("%w: %s", domain.ErrDepositCurrencyMismatch, results[0].Status)
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
		return domain.Deposit{}, fmt.Errorf("%w: %s", deposits.ErrDepositConflict, results[0].Status)
	default:
		return domain.Deposit{}, fmt.Errorf(
			"%w: creating transfer %s: %s",
			deposits.ErrDepositStoreUnavailable,
			deposit.TransferID,
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
				deposits.ErrDepositStoreUnavailable,
			)
		}
		return accountID, nil
	}

	externalID, err := deterministicExternalID(
		"ledger.bank-cash.v1",
		strconv.FormatUint(uint64(ledger), 10),
	)
	if err != nil {
		return tigerbeetle_go.Uint128{}, fmt.Errorf("deriving bank cash account ID: %w", err)
	}

	accountID := tigerbeetle_go.BytesToUint128(externalID.Bytes())
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
			deposits.ErrDepositStoreUnavailable,
			err,
		)
	}
	if len(results) != 1 {
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: expected 1 bank cash create result, got %d",
			deposits.ErrDepositStoreUnavailable,
			len(results),
		)
	}

	//nolint:exhaustive // Unknown statuses fall through to the store-unavailable error.
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
			deposits.ErrDepositConflict,
			ledger,
			results[0].Status,
		)
	default:
		return tigerbeetle_go.Uint128{}, fmt.Errorf(
			"%w: creating bank cash account for ledger %d: %s",
			deposits.ErrDepositStoreUnavailable,
			ledger,
			results[0].Status,
		)
	}
}

func (r *Deposits) verifyExistingTransfer(expected tigerbeetle_go.Transfer) error {
	existing, err := r.client.LookupTransfers([]tigerbeetle_go.Uint128{expected.ID})
	if err != nil {
		return fmt.Errorf("%w: %w", deposits.ErrDepositStoreUnavailable, err)
	}
	if len(existing) != 1 {
		return fmt.Errorf("%w: existing transfer %s not found", deposits.ErrDepositConflict, expected.ID)
	}
	if !sameTransfer(existing[0], expected) {
		return fmt.Errorf("%w: transfer %s", deposits.ErrDepositConflict, expected.ID)
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

func deterministicExternalID(namespace string, value string) (domain.ExternalID, error) {
	digest := sha256.Sum256([]byte(namespace + ":" + value))
	return domain.ParseExternalID("0x" + hex.EncodeToString(digest[:16]))
}
