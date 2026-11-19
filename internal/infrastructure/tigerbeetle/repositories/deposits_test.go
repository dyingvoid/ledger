package repositories

import (
	"context"
	"errors"
	"testing"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/deposits"
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

func testDeposit(t *testing.T, clientAccountID string, bankAccountID string, transferID string) domain.Deposit {
	t.Helper()

	deposit, err := domain.NewDeposit(
		testID(t, transferID),
		2500,
		domain.CurrencyInfo{
			Code:       testUSD,
			Name:       "US Dollar",
			Ledger:     1,
			AssetScale: 2,
		},
		domain.Account{
			ID:       testID(t, clientAccountID),
			UserID:   7,
			Name:     "Client",
			Type:     domain.AccountTypeClientDeposit,
			Currency: testUSD,
		},
		domain.Account{
			ID:       testID(t, bankAccountID),
			UserID:   1,
			Name:     "Bank Cash",
			Type:     domain.AccountTypeBankCash,
			Currency: testUSD,
		},
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}
	return deposit
}

func TestCreateDepositMapsTransfer(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		transferResults: []tigerbeetle_go.CreateTransferResult{{Status: tigerbeetle_go.TransferCreated}},
	}
	repository := NewDeposits(client)
	deposit := testDeposit(t, "42", "43", "99")

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
	if client.calls != 0 {
		t.Fatalf("CreateAccounts() calls = %d, want 0", client.calls)
	}
	if len(client.transfers) != 1 {
		t.Fatalf("transfers = %d, want 1", len(client.transfers))
	}

	transfer := client.transfers[0]
	if transfer.ID != toTigerbeetleID(deposit.ID) {
		t.Fatalf("transfer ID = %s, want %s", transfer.ID, deposit.ID)
	}
	if transfer.DebitAccountID != toTigerbeetleID(deposit.BankAccount.ID) {
		t.Fatalf("debit account = %s, want %s", transfer.DebitAccountID, deposit.BankAccount.ID)
	}
	if transfer.CreditAccountID != toTigerbeetleID(deposit.ClientAccount.ID) {
		t.Fatalf("credit account = %s, want %s", transfer.CreditAccountID, deposit.ClientAccount.ID)
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

func TestCreateDepositMapsStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  tigerbeetle_go.CreateTransferStatus
		wantErr error
	}{
		{
			name:    "bank account missing in ledger",
			status:  tigerbeetle_go.TransferDebitAccountNotFound,
			wantErr: domain_errors.ErrDepositBankAccountNotFound,
		},
		{
			name:    "client account missing in ledger",
			status:  tigerbeetle_go.TransferCreditAccountNotFound,
			wantErr: domain_errors.ErrDepositClientAccountNotFound,
		},
		{
			name:    "accounts have different ledgers",
			status:  tigerbeetle_go.TransferAccountsMustHaveTheSameLedger,
			wantErr: domain_errors.ErrDepositCurrencyMismatch,
		},
		{
			name:    "existing transfer with different amount",
			status:  tigerbeetle_go.TransferExistsWithDifferentAmount,
			wantErr: domain_errors.ErrDepositConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeClient{
				transferResults: []tigerbeetle_go.CreateTransferResult{{Status: test.status}},
			}
			repository := NewDeposits(client)

			_, err := repository.Create(context.Background(), testDeposit(t, "42", "43", "99"))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCreateDepositTreatsExistingTransferAsSuccess(t *testing.T) {
	t.Parallel()

	deposit := testDeposit(t, "42", "43", "99")
	expected := tigerbeetle_go.Transfer{
		ID:              toTigerbeetleID(deposit.ID),
		DebitAccountID:  toTigerbeetleID(deposit.BankAccount.ID),
		CreditAccountID: toTigerbeetleID(deposit.ClientAccount.ID),
		Amount:          tigerbeetle_go.ToUint128(2500),
		UserData128:     toTigerbeetleID(deposit.ClientAccount.ID),
		Ledger:          1,
		Code:            depositTransferCode,
	}
	client := &fakeClient{
		transferResults: []tigerbeetle_go.CreateTransferResult{{Status: tigerbeetle_go.TransferExists}},
		lookupTransfers: []tigerbeetle_go.Transfer{expected},
	}
	repository := NewDeposits(client)

	_, err := repository.Create(context.Background(), deposit)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestCreateDepositRejectsConflictingExistingTransfer(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		transferResults: []tigerbeetle_go.CreateTransferResult{{Status: tigerbeetle_go.TransferExists}},
		lookupTransfers: []tigerbeetle_go.Transfer{{
			ID:     toTigerbeetleID(testID(t, "99")),
			Amount: tigerbeetle_go.ToUint128(1),
		}},
	}
	repository := NewDeposits(client)

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "43", "99"))
	if !errors.Is(err, domain_errors.ErrDepositConflict) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrDepositConflict)
	}
}

func TestCreateDepositLedgerUnavailable(t *testing.T) {
	t.Parallel()

	clientErr := errors.New("connection lost")
	client := &fakeClient{transferErr: clientErr}
	repository := NewDeposits(client)

	_, err := repository.Create(context.Background(), testDeposit(t, "42", "43", "99"))
	if !errors.Is(err, deposits.ErrLedgerUnavailable) {
		t.Fatalf("Create() error = %v, want %v", err, deposits.ErrLedgerUnavailable)
	}
}
