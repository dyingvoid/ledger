package repositories

import (
	"context"
	"errors"
	"testing"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const testUSD = "USD"

type fakeClient struct {
	results  []tigerbeetle_go.CreateAccountResult
	err      error
	accounts []tigerbeetle_go.Account
	calls    int

	transferResults   []tigerbeetle_go.CreateTransferResult
	transferErr       error
	transfers         []tigerbeetle_go.Transfer
	transferCalls     int
	lookupAccounts    []tigerbeetle_go.Account
	lookupAccountIDs  []tigerbeetle_go.Uint128
	lookupTransfers   []tigerbeetle_go.Transfer
	lookupTransferIDs []tigerbeetle_go.Uint128
}

func (f *fakeClient) CreateAccounts(accounts []tigerbeetle_go.Account) ([]tigerbeetle_go.CreateAccountResult, error) {
	f.calls++
	f.accounts = accounts
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

func (f *fakeClient) CreateTransfers(transfers []tigerbeetle_go.Transfer) ([]tigerbeetle_go.CreateTransferResult, error) {
	index := f.transferCalls
	f.transferCalls++
	f.transfers = transfers
	if f.transferErr != nil {
		return nil, f.transferErr
	}
	if index < len(f.transferResults) {
		return f.transferResults[index : index+1], nil
	}
	return f.transferResults, nil
}

func (f *fakeClient) LookupAccounts(ids []tigerbeetle_go.Uint128) ([]tigerbeetle_go.Account, error) {
	f.lookupAccountIDs = ids
	return f.lookupAccounts, nil
}

func (f *fakeClient) LookupTransfers(ids []tigerbeetle_go.Uint128) ([]tigerbeetle_go.Transfer, error) {
	f.lookupTransferIDs = ids
	return f.lookupTransfers, nil
}

func (f *fakeClient) GetAccountTransfers(tigerbeetle_go.AccountFilter) ([]tigerbeetle_go.Transfer, error) {
	return nil, nil
}

func (f *fakeClient) GetAccountBalances(tigerbeetle_go.AccountFilter) ([]tigerbeetle_go.AccountBalance, error) {
	return nil, nil
}

func (f *fakeClient) QueryAccounts(tigerbeetle_go.QueryFilter) ([]tigerbeetle_go.Account, error) {
	return nil, nil
}

func (f *fakeClient) QueryTransfers(tigerbeetle_go.QueryFilter) ([]tigerbeetle_go.Transfer, error) {
	return nil, nil
}

func (f *fakeClient) GetChangeEvents(tigerbeetle_go.ChangeEventsFilter) ([]tigerbeetle_go.ChangeEvent, error) {
	return nil, nil
}

func (f *fakeClient) Nop() error {
	return nil
}

func (f *fakeClient) Close() {}

type fakeCurrencies struct {
	info domain.CurrencyInfo
	err  error
}

func (f *fakeCurrencies) Get(_ context.Context, _ string) (domain.CurrencyInfo, error) {
	if f.err != nil {
		return domain.CurrencyInfo{}, f.err
	}
	return f.info, nil
}

func testAccount(t *testing.T, accountType domain.AccountType) domain.Account {
	t.Helper()

	id, err := domain.ParseID("123456789")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	return domain.Account{
		ID:       id,
		UserID:   7,
		Name:     "Daily",
		Type:     accountType,
		Currency: testUSD,
	}
}

func TestCreateAccountMapsFields(t *testing.T) {
	t.Parallel()

	client := &fakeClient{results: []tigerbeetle_go.CreateAccountResult{
		{Status: tigerbeetle_go.AccountCreated, Timestamp: 1_000_000_000},
	}}
	repository := NewAccounts(client, &fakeCurrencies{
		info: domain.CurrencyInfo{Code: testUSD, Ledger: 1, AssetScale: 2},
	})

	account := testAccount(t, 1000)
	created, err := repository.Create(context.Background(), account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if client.calls != 1 {
		t.Fatalf("CreateAccounts() calls = %d, want 1", client.calls)
	}
	if len(client.accounts) != 1 {
		t.Fatalf("CreateAccounts() accounts = %d, want 1", len(client.accounts))
	}

	sent := client.accounts[0]
	if sent.ID != toTigerbeetleID(account.ID) {
		t.Fatalf("ID = %s, want %s", sent.ID, account.ID)
	}
	if sent.UserData64 != 7 {
		t.Fatalf("UserData64 = %d, want 7", sent.UserData64)
	}
	if sent.Ledger != 1 {
		t.Fatalf("Ledger = %d, want 1", sent.Ledger)
	}
	if sent.Code != 1000 {
		t.Fatalf("Code = %d, want 1000", sent.Code)
	}
	if !sent.AccountFlags().CreditsMustNotExceedDebits {
		t.Fatalf("CreditsMustNotExceedDebits = false, want true")
	}
	if sent.AccountFlags().DebitsMustNotExceedCredits {
		t.Fatalf("DebitsMustNotExceedCredits = true, want false")
	}
	if created.ID != account.ID {
		t.Fatalf("ID = %s, want %s", created.ID, account.ID)
	}
}

func TestCreateAccountFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		accountType          domain.AccountType
		wantCreditsNotExceed bool
		wantDebitsNotExceed  bool
	}{
		{name: "asset", accountType: 1000, wantCreditsNotExceed: true},
		{name: "liability", accountType: 2100, wantDebitsNotExceed: true},
		{name: "equity", accountType: 3100, wantDebitsNotExceed: true},
		{name: "revenue", accountType: 4100, wantDebitsNotExceed: true},
		{name: "expense", accountType: 5100, wantCreditsNotExceed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeClient{results: []tigerbeetle_go.CreateAccountResult{
				{Status: tigerbeetle_go.AccountCreated},
			}}
			repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

			if _, err := repository.Create(context.Background(), testAccount(t, test.accountType)); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			flags := client.accounts[0].AccountFlags()
			if flags.CreditsMustNotExceedDebits != test.wantCreditsNotExceed {
				t.Fatalf(
					"CreditsMustNotExceedDebits = %t, want %t",
					flags.CreditsMustNotExceedDebits,
					test.wantCreditsNotExceed,
				)
			}
			if flags.DebitsMustNotExceedCredits != test.wantDebitsNotExceed {
				t.Fatalf(
					"DebitsMustNotExceedCredits = %t, want %t",
					flags.DebitsMustNotExceedCredits,
					test.wantDebitsNotExceed,
				)
			}
		})
	}
}

func TestCreateAccountResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  tigerbeetle_go.CreateAccountStatus
		wantErr error
	}{
		{name: "created", status: tigerbeetle_go.AccountCreated},
		{name: "exists", status: tigerbeetle_go.AccountExists},
		{name: "different flags", status: tigerbeetle_go.AccountExistsWithDifferentFlags, wantErr: accounts.ErrAccountConflict},
		{name: "different user data 128", status: tigerbeetle_go.AccountExistsWithDifferentUserData128, wantErr: accounts.ErrAccountConflict},
		{name: "different user data 64", status: tigerbeetle_go.AccountExistsWithDifferentUserData64, wantErr: accounts.ErrAccountConflict},
		{name: "different user data 32", status: tigerbeetle_go.AccountExistsWithDifferentUserData32, wantErr: accounts.ErrAccountConflict},
		{name: "different ledger", status: tigerbeetle_go.AccountExistsWithDifferentLedger, wantErr: accounts.ErrAccountConflict},
		{name: "different code", status: tigerbeetle_go.AccountExistsWithDifferentCode, wantErr: accounts.ErrAccountConflict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeClient{results: []tigerbeetle_go.CreateAccountResult{{Status: test.status}}}
			repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

			_, err := repository.Create(context.Background(), testAccount(t, 1000))
			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("Create() error = %v", err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCreateAccountUnexpectedStatus(t *testing.T) {
	t.Parallel()

	client := &fakeClient{results: []tigerbeetle_go.CreateAccountResult{
		{Status: tigerbeetle_go.AccountIDMustNotBeZero},
	}}
	repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

	if _, err := repository.Create(context.Background(), testAccount(t, 1000)); err == nil {
		t.Fatal("Create() expected error")
	}
}

func TestCreateAccountClientError(t *testing.T) {
	t.Parallel()

	clientErr := errors.New("connection lost")
	client := &fakeClient{err: clientErr}
	repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

	_, err := repository.Create(context.Background(), testAccount(t, 1000))
	if !errors.Is(err, accounts.ErrAccountStoreUnavailable) {
		t.Fatalf("Create() error = %v, want %v", err, accounts.ErrAccountStoreUnavailable)
	}
}

func TestCreateAccountRequiresID(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

	account := testAccount(t, 1000)
	account.ID = domain.ID{}

	_, err := repository.Create(context.Background(), account)
	if !errors.Is(err, domain_errors.ErrAccountIDRequired) {
		t.Fatalf("Create() error = %v, want %v", err, domain_errors.ErrAccountIDRequired)
	}
	if client.calls != 0 {
		t.Fatalf("CreateAccounts() calls = %d, want 0", client.calls)
	}
}

func TestCreateAccountCurrencyNotFound(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := NewAccounts(client, &fakeCurrencies{err: accounts.ErrCurrencyNotFound})

	_, err := repository.Create(context.Background(), testAccount(t, 1000))
	if !errors.Is(err, accounts.ErrCurrencyNotFound) {
		t.Fatalf("Create() error = %v, want %v", err, accounts.ErrCurrencyNotFound)
	}
	if client.calls != 0 {
		t.Fatalf("CreateAccounts() calls = %d, want 0", client.calls)
	}
}
