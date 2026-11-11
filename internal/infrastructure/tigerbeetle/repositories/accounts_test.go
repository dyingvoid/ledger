package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
)

const testUSD domain.Currency = "USD"

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

func (f *fakeCurrencies) Get(_ context.Context, _ domain.Currency) (domain.CurrencyInfo, error) {
	if f.err != nil {
		return domain.CurrencyInfo{}, f.err
	}
	return f.info, nil
}

func testAccount(t *testing.T, accountType domain.AccountType) domain.Account {
	t.Helper()

	externalID, err := domain.ParseExternalID("123456789")
	if err != nil {
		t.Fatalf("ParseExternalID() error = %v", err)
	}

	return domain.Account{
		ExternalID: externalID,
		UserID:     7,
		Name:       "Daily",
		Type:       accountType,
		Currency:   testUSD,
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
	if sent.ID != tigerbeetle_go.BytesToUint128(account.ExternalID.Bytes()) {
		t.Fatalf("ID = %s, want %s", sent.ID, account.ExternalID)
	}
	if sent.ID != tigerbeetle_go.BigIntToUint128(account.ExternalID.BigInt()) {
		t.Fatalf("ID = %s, want big-int encoding %s", sent.ID, account.ExternalID)
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
	if !created.CreatedAt.Equal(time.Unix(0, 1_000_000_000).UTC()) {
		t.Fatalf("CreatedAt = %s, want %s", created.CreatedAt, time.Unix(0, 1_000_000_000).UTC())
	}
	if created.ID != 0 {
		t.Fatalf("ID = %d, want 0", created.ID)
	}
}

func TestCreateAccountCreditFlags(t *testing.T) {
	t.Parallel()

	client := &fakeClient{results: []tigerbeetle_go.CreateAccountResult{
		{Status: tigerbeetle_go.AccountCreated},
	}}
	repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

	if _, err := repository.Create(context.Background(), testAccount(t, 2100)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	flags := client.accounts[0].AccountFlags()
	if !flags.DebitsMustNotExceedCredits {
		t.Fatalf("DebitsMustNotExceedCredits = false, want true")
	}
	if flags.CreditsMustNotExceedDebits {
		t.Fatalf("CreditsMustNotExceedDebits = true, want false")
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
		{name: "different flags", status: tigerbeetle_go.AccountExistsWithDifferentFlags, wantErr: accounts.ErrAccountIDConflict},
		{name: "different user data 128", status: tigerbeetle_go.AccountExistsWithDifferentUserData128, wantErr: accounts.ErrAccountIDConflict},
		{name: "different user data 64", status: tigerbeetle_go.AccountExistsWithDifferentUserData64, wantErr: accounts.ErrAccountIDConflict},
		{name: "different user data 32", status: tigerbeetle_go.AccountExistsWithDifferentUserData32, wantErr: accounts.ErrAccountIDConflict},
		{name: "different ledger", status: tigerbeetle_go.AccountExistsWithDifferentLedger, wantErr: accounts.ErrAccountIDConflict},
		{name: "different code", status: tigerbeetle_go.AccountExistsWithDifferentCode, wantErr: accounts.ErrAccountIDConflict},
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

func TestCreateAccountRequiresExternalID(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := NewAccounts(client, &fakeCurrencies{info: domain.CurrencyInfo{Ledger: 1}})

	account := testAccount(t, 1000)
	account.ExternalID = domain.ExternalID{}

	_, err := repository.Create(context.Background(), account)
	if !errors.Is(err, accounts.ErrExternalIDRequired) {
		t.Fatalf("Create() error = %v, want %v", err, accounts.ErrExternalIDRequired)
	}
	if client.calls != 0 {
		t.Fatalf("CreateAccounts() calls = %d, want 0", client.calls)
	}
}

func TestCreateAccountCurrencyNotConfigured(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	repository := NewAccounts(client, &fakeCurrencies{err: accounts.ErrCurrencyNotFound})

	_, err := repository.Create(context.Background(), testAccount(t, 1000))
	if !errors.Is(err, accounts.ErrCurrencyNotConfigured) {
		t.Fatalf("Create() error = %v, want %v", err, accounts.ErrCurrencyNotConfigured)
	}
	if client.calls != 0 {
		t.Fatalf("CreateAccounts() calls = %d, want 0", client.calls)
	}
}
