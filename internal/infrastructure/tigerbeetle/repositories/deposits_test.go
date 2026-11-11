package repositories

import (
	"context"
	"errors"
	"strconv"
	"testing"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

func testID(t *testing.T, value string) domain.ID {
	t.Helper()

	id, err := domain.ParseID(value)
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}
	return id
}

func testDeposit(t *testing.T, accountID string, transferID string) domain.Deposit {
	t.Helper()

	deposit, err := domain.NewDeposit(
		testID(t, transferID),
		testID(t, accountID),
		2500,
		testUSD,
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}
	return deposit
}

func testDepositRepository(client *fakeClient, info domain.CurrencyInfo) *Deposits {
	return NewDeposits(client, &fakeCurrencies{info: info})
}

func testBankCashID(t *testing.T, ledger uint32) tigerbeetle_go.Uint128 {
	t.Helper()

	id, err := deterministicID(
		"ledger.bank-cash.v1",
		strconv.FormatUint(uint64(ledger), 10),
	)
	if err != nil {
		t.Fatalf("deterministicID() error = %v", err)
	}
	return toTigerbeetleID(id)
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
	if created.ID != deposit.ID {
		t.Fatalf("ID = %s, want %s", created.ID, deposit.ID)
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
	if transfer.ID != toTigerbeetleID(deposit.ID) {
		t.Fatalf("transfer ID = %s, want %s", transfer.ID, deposit.ID)
	}
	if transfer.DebitAccountID != testBankCashID(t, 1) {
		t.Fatalf("debit account = %s, want bank cash account", transfer.DebitAccountID)
	}
	if transfer.CreditAccountID != toTigerbeetleID(deposit.AccountID) {
		t.Fatalf("credit account = %s, want %s", transfer.CreditAccountID, deposit.AccountID)
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

func TestCreateDepositRejectsUnknownAccount(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, domain_errors.ErrDepositAccountNotFound) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrDepositAccountNotFound)
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
	if !errors.Is(err, domain_errors.ErrDepositAccountNotClient) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrDepositAccountNotClient)
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
	if !errors.Is(err, domain_errors.ErrDepositCurrencyMismatch) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrDepositCurrencyMismatch)
	}
}

func TestCreateDepositTreatsExistingTransferAsSuccess(t *testing.T) {
	t.Parallel()

	expected := tigerbeetle_go.Transfer{
		ID:              toTigerbeetleID(testID(t, "99")),
		DebitAccountID:  testBankCashID(t, 1),
		CreditAccountID: toTigerbeetleID(testID(t, "42")),
		Amount:          tigerbeetle_go.ToUint128(2500),
		UserData128:     toTigerbeetleID(testID(t, "42")),
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
			ID:     toTigerbeetleID(testID(t, "99")),
			Amount: tigerbeetle_go.ToUint128(1),
		}},
	}
	repository := testDepositRepository(client, domain.CurrencyInfo{Code: testUSD, Ledger: 1})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, domain_errors.ErrDepositConflict) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrDepositConflict)
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

func TestCreateDepositCurrencyNotFound(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := NewDeposits(client, &fakeCurrencies{err: domain_errors.ErrAccountCurrencyRequired})

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "99"))
	if !errors.Is(err, domain_errors.ErrAccountCurrencyRequired) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrAccountCurrencyRequired)
	}
}
