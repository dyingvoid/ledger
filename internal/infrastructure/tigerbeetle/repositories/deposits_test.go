package repositories

import (
	"context"
	"errors"
	"strconv"
	"testing"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/domain"
)

func testExternalID(t *testing.T, value string) domain.ExternalID {
	t.Helper()

	id, err := domain.ParseExternalID(value)
	if err != nil {
		t.Fatalf("ParseExternalID() error = %v", err)
	}
	return id
}

func testDeposit(t *testing.T, accountID string, transferID string) domain.Deposit {
	t.Helper()

	deposit, err := domain.NewDeposit(
		0,
		testExternalID(t, accountID),
		2500,
		testUSD,
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}
	deposit.TransferID = testExternalID(t, transferID)
	return deposit
}

func testDepositRepository(client *fakeClient, info domain.CurrencyInfo) *Deposits {
	return NewDeposits(client, &fakeCurrencies{info: info})
}

func testBankCashID(t *testing.T, ledger uint32) tigerbeetle_go.Uint128 {
	t.Helper()

	externalID, err := deterministicExternalID(
		"ledger.bank-cash.v1",
		strconv.FormatUint(uint64(ledger), 10),
	)
	if err != nil {
		t.Fatalf("deterministicExternalID() error = %v", err)
	}
	return tigerbeetle_go.BytesToUint128(externalID.Bytes())
}

func TestCreateDepositMapsTransfer(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		results: []tigerbeetle_go.CreateAccountResult{{Status: tigerbeetle_go.AccountCreated}},
		lookupAccounts: []tigerbeetle_go.Account{{
			Ledger: 1,
			Code:   uint16(domain.AccountTypeClientDeposit),
		}},
		transferResults: []tigerbeetle_go.CreateTransferResult{{Status: tigerbeetle_go.TransferCreated}},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})
	deposit := testDeposit(t, "42", "99")

	created, err := repository.Create(context.Background(), deposit)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.TransferID != deposit.TransferID {
		t.Fatalf("TransferID = %s, want %s", created.TransferID, deposit.TransferID)
	}
	if client.transferCalls != 1 {
		t.Fatalf("CreateTransfers() calls = %d, want 1", client.transferCalls)
	}
	if client.calls != 1 {
		t.Fatalf("CreateAccounts() calls = %d, want 1", client.calls)
	}
	if len(client.transfers) != 1 {
		t.Fatalf("transfers = %d, want 1", len(client.transfers))
	}

	transfer := client.transfers[0]
	if transfer.ID != tigerbeetle_go.BytesToUint128(deposit.TransferID.Bytes()) {
		t.Fatalf("transfer ID = %s, want %s", transfer.ID, deposit.TransferID)
	}
	if transfer.DebitAccountID != testBankCashID(t, 1) {
		t.Fatalf("debit account = %s, want bank cash account", transfer.DebitAccountID)
	}
	if transfer.CreditAccountID != tigerbeetle_go.BytesToUint128(deposit.ExternalAccountID.Bytes()) {
		t.Fatalf("credit account = %s, want %s", transfer.CreditAccountID, deposit.ExternalAccountID)
	}
	if transfer.Amount != tigerbeetle_go.ToUint128(2500) {
		t.Fatalf("amount = %s, want 2500", transfer.Amount)
	}
	if transfer.Ledger != 1 {
		t.Fatalf("ledger = %d, want 1", transfer.Ledger)
	}
	if transfer.Code != depositTransferCode {
		t.Fatalf("code = %d, want %d", transfer.Code, depositTransferCode)
	}
}

func TestCreateDepositRejectsMissingExternalAccount(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})
	deposit := testDeposit(t, "42", "99")
	deposit.ExternalAccountID = domain.ExternalID{}

	_, err := repository.Create(context.Background(), deposit)
	if !errors.Is(err, domain.ErrDepositAccountRequired) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrDepositAccountRequired)
	}
	if client.transferCalls != 0 {
		t.Fatalf("CreateTransfers() calls = %d, want 0", client.transferCalls)
	}
}

func TestCreateDepositRejectsMissingTransferID(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})
	deposit := testDeposit(t, "42", "99")
	deposit.TransferID = domain.ExternalID{}

	_, err := repository.Create(context.Background(), deposit)
	if !errors.Is(err, deposits.ErrTransferIDRequired) {
		t.Fatalf("Create() error = %v, want %v", err, deposits.ErrTransferIDRequired)
	}
}

func TestCreateDepositRejectsUnknownAccount(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, domain.ErrDepositAccountNotFound) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrDepositAccountNotFound)
	}
	if client.calls != 0 {
		t.Fatalf("CreateAccounts() calls = %d, want 0", client.calls)
	}
}

func TestCreateDepositRejectsNonClientAccount(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		lookupAccounts: []tigerbeetle_go.Account{{
			Ledger: 1,
			Code:   uint16(domain.AccountTypeBankCash),
		}},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, domain.ErrDepositAccountNotClient) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrDepositAccountNotClient)
	}
}

func TestCreateDepositRejectsCurrencyMismatch(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		lookupAccounts: []tigerbeetle_go.Account{{
			Ledger: 2,
			Code:   uint16(domain.AccountTypeClientDeposit),
		}},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, domain.ErrDepositCurrencyMismatch) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrDepositCurrencyMismatch)
	}
}

func TestCreateDepositTreatsExistingTransferAsSuccess(t *testing.T) {
	t.Parallel()

	expected := tigerbeetle_go.Transfer{
		ID:              tigerbeetle_go.BytesToUint128(testExternalID(t, "99").Bytes()),
		DebitAccountID:  testBankCashID(t, 1),
		CreditAccountID: tigerbeetle_go.BytesToUint128(testExternalID(t, "42").Bytes()),
		Amount:          tigerbeetle_go.ToUint128(2500),
		UserData128:     tigerbeetle_go.BytesToUint128(testExternalID(t, "42").Bytes()),
		Ledger:          1,
		Code:            depositTransferCode,
	}
	client := &fakeClient{
		results: []tigerbeetle_go.CreateAccountResult{{Status: tigerbeetle_go.AccountExists}},
		lookupAccounts: []tigerbeetle_go.Account{{
			Ledger: 1,
			Code:   uint16(domain.AccountTypeClientDeposit),
		}},
		transferResults: []tigerbeetle_go.CreateTransferResult{{Status: tigerbeetle_go.TransferExists}},
		lookupTransfers: []tigerbeetle_go.Transfer{expected},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestCreateDepositRejectsConflictingExistingTransfer(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		results: []tigerbeetle_go.CreateAccountResult{{Status: tigerbeetle_go.AccountExists}},
		lookupAccounts: []tigerbeetle_go.Account{{
			Ledger: 1,
			Code:   uint16(domain.AccountTypeClientDeposit),
		}},
		transferResults: []tigerbeetle_go.CreateTransferResult{{Status: tigerbeetle_go.TransferExists}},
		lookupTransfers: []tigerbeetle_go.Transfer{{
			ID:     tigerbeetle_go.BytesToUint128(testExternalID(t, "99").Bytes()),
			Amount: tigerbeetle_go.ToUint128(1),
		}},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, deposits.ErrDepositConflict) {
		t.Fatalf("Create() error = %v, want %v", err, deposits.ErrDepositConflict)
	}
}

func TestCreateDepositEnsuresBankCashAccountOnce(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		results: []tigerbeetle_go.CreateAccountResult{{Status: tigerbeetle_go.AccountCreated}},
		lookupAccounts: []tigerbeetle_go.Account{{
			Ledger: 1,
			Code:   uint16(domain.AccountTypeClientDeposit),
		}},
		transferResults: []tigerbeetle_go.CreateTransferResult{
			{Status: tigerbeetle_go.TransferCreated},
			{Status: tigerbeetle_go.TransferCreated},
		},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	if _, err := repository.Create(context.Background(), testDeposit(t, "42", "99")); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := repository.Create(context.Background(), testDeposit(t, "42", "100")); err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("CreateAccounts() calls = %d, want 1", client.calls)
	}
	if client.transferCalls != 2 {
		t.Fatalf("CreateTransfers() calls = %d, want 2", client.transferCalls)
	}
}

func TestCreateDepositCurrencyNotConfigured(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := NewDeposits(client, &fakeCurrencies{err: accounts.ErrCurrencyNotFound})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, accounts.ErrCurrencyNotConfigured) {
		t.Fatalf("Create() error = %v, want %v", err, accounts.ErrCurrencyNotConfigured)
	}
}
