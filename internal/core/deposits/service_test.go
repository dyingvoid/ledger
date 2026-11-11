package deposits

import (
	"context"
	"errors"
	"testing"

	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const testCurrency = "USD"

type fakeAccountRepository struct {
	client      domain.Account
	bank        domain.Account
	clientErr   error
	bankErr     error
	clientCalls int
	bankCalls   int
}

func (f *fakeAccountRepository) Get(_ context.Context, _ domain.ID) (domain.Account, error) {
	f.clientCalls++
	if f.clientErr != nil {
		return domain.Account{}, f.clientErr
	}
	return f.client, nil
}

func (f *fakeAccountRepository) GetBankCash(_ context.Context, _ string) (domain.Account, error) {
	f.bankCalls++
	if f.bankErr != nil {
		return domain.Account{}, f.bankErr
	}
	return f.bank, nil
}

type fakeCurrencyRepository struct {
	currency domain.CurrencyInfo
	err      error
	calls    int
}

func (f *fakeCurrencyRepository) Get(_ context.Context, _ string) (domain.CurrencyInfo, error) {
	f.calls++
	if f.err != nil {
		return domain.CurrencyInfo{}, f.err
	}
	return f.currency, nil
}

type fakeLedgerRepository struct {
	createErr   error
	created     domain.Deposit
	createCalls int
}

func (f *fakeLedgerRepository) Create(_ context.Context, deposit domain.Deposit) (domain.Deposit, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.Deposit{}, f.createErr
	}
	f.created = deposit
	return deposit, nil
}

func testID(t *testing.T, value string) domain.ID {
	t.Helper()

	id, err := domain.ParseID(value)
	if err != nil {
		t.Fatalf("ParseID(%q) error = %v", value, err)
	}
	return id
}

func testService(
	t *testing.T,
	accounts *fakeAccountRepository,
	currencies *fakeCurrencyRepository,
	ledger *fakeLedgerRepository,
) *Service {
	t.Helper()

	if accounts.client.ID.IsZero() {
		accounts.client = domain.Account{
			ID:       testID(t, "1"),
			UserID:   7,
			Name:     "Client",
			Type:     domain.AccountTypeClientDeposit,
			Currency: testCurrency,
		}
	}
	if accounts.bank.ID.IsZero() {
		accounts.bank = domain.Account{
			ID:       testID(t, "2"),
			UserID:   1,
			Name:     "Bank Cash",
			Type:     domain.AccountTypeBankCash,
			Currency: testCurrency,
		}
	}
	if currencies.currency.Code == "" {
		currencies.currency = domain.CurrencyInfo{
			Code:       testCurrency,
			Name:       "US Dollar",
			Ledger:     1,
			AssetScale: 2,
		}
	}

	return NewService(accounts, currencies, ledger)
}

func testDepositParams(t *testing.T) DepositParams {
	t.Helper()

	return DepositParams{
		ID:          testID(t, "9"),
		AccountID:   testID(t, "1"),
		AmountMinor: 2500,
		Currency:    testCurrency,
	}
}

func TestDepositCreatesValidDeposit(t *testing.T) {
	t.Parallel()

	accounts := &fakeAccountRepository{}
	currencies := &fakeCurrencyRepository{}
	ledger := &fakeLedgerRepository{}
	service := testService(t, accounts, currencies, ledger)

	deposit, err := service.Deposit(context.Background(), testDepositParams(t))
	if err != nil {
		t.Fatalf("Deposit() error = %v", err)
	}
	if deposit.ID.String() != "9" {
		t.Fatalf("ID = %s, want 9", deposit.ID)
	}
	if ledger.createCalls != 1 {
		t.Fatalf("Create() calls = %d, want 1", ledger.createCalls)
	}
	if ledger.created.ClientAccount.ID.String() != "1" {
		t.Fatalf("ClientAccount.ID = %s, want 1", ledger.created.ClientAccount.ID)
	}
	if ledger.created.BankAccount.ID.String() != "2" {
		t.Fatalf("BankAccount.ID = %s, want 2", ledger.created.BankAccount.ID)
	}
	if ledger.created.Currency.Ledger != 1 {
		t.Fatalf("Currency.Ledger = %d, want 1", ledger.created.Currency.Ledger)
	}
	if ledger.created.AmountMinor != 2500 {
		t.Fatalf("AmountMinor = %d, want 2500", ledger.created.AmountMinor)
	}
}

func TestDepositRejectsInvalidParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*DepositParams)
		want   error
	}{
		{
			name:   "rejects missing deposit ID",
			mutate: func(p *DepositParams) { p.ID = domain.ID{} },
			want:   domain_errors.ErrDepositIDRequired,
		},
		{
			name:   "rejects zero amount",
			mutate: func(p *DepositParams) { p.AmountMinor = 0 },
			want:   domain_errors.ErrDepositAmountInvalid,
		},
		{
			name:   "rejects blank currency",
			mutate: func(p *DepositParams) { p.Currency = " " },
			want:   domain_errors.ErrDepositCurrencyRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			accounts := &fakeAccountRepository{}
			currencies := &fakeCurrencyRepository{}
			ledger := &fakeLedgerRepository{}
			service := testService(t, accounts, currencies, ledger)
			params := testDepositParams(t)
			test.mutate(&params)

			_, err := service.Deposit(context.Background(), params)
			if !errors.Is(err, test.want) {
				t.Fatalf("Deposit() error = %v, want %v", err, test.want)
			}
			if accounts.clientCalls != 0 || currencies.calls != 0 || ledger.createCalls != 0 {
				t.Fatalf(
					"unexpected calls: accounts=%d currencies=%d ledger=%d",
					accounts.clientCalls,
					currencies.calls,
					ledger.createCalls,
				)
			}
		})
	}
}

func TestDepositRejectsUnsupportedClientAccount(t *testing.T) {
	t.Parallel()

	accounts := &fakeAccountRepository{}
	currencies := &fakeCurrencyRepository{}
	ledger := &fakeLedgerRepository{}
	service := testService(t, accounts, currencies, ledger)
	accounts.client.Type = domain.AccountTypeBankCash

	_, err := service.Deposit(context.Background(), testDepositParams(t))
	if !errors.Is(err, domain_errors.ErrDepositClientAccountTypeUnsupported) {
		t.Fatalf("Deposit() error = %v, want %v", err, domain_errors.ErrDepositClientAccountTypeUnsupported)
	}
	if ledger.createCalls != 0 {
		t.Fatalf("Create() calls = %d, want 0", ledger.createCalls)
	}
}

func TestDepositWrapsAccountLookupErrors(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("account lookup failed")

	t.Run("client account", func(t *testing.T) {
		t.Parallel()

		accounts := &fakeAccountRepository{clientErr: lookupErr}
		currencies := &fakeCurrencyRepository{}
		ledger := &fakeLedgerRepository{}
		service := testService(t, accounts, currencies, ledger)

		_, err := service.Deposit(context.Background(), testDepositParams(t))
		if !errors.Is(err, lookupErr) {
			t.Fatalf("Deposit() error = %v, want %v", err, lookupErr)
		}
		if currencies.calls != 0 || ledger.createCalls != 0 {
			t.Fatalf("unexpected calls: currencies=%d ledger=%d", currencies.calls, ledger.createCalls)
		}
	})

	t.Run("bank cash account", func(t *testing.T) {
		t.Parallel()

		accounts := &fakeAccountRepository{bankErr: lookupErr}
		currencies := &fakeCurrencyRepository{}
		ledger := &fakeLedgerRepository{}
		service := testService(t, accounts, currencies, ledger)

		_, err := service.Deposit(context.Background(), testDepositParams(t))
		if !errors.Is(err, lookupErr) {
			t.Fatalf("Deposit() error = %v, want %v", err, lookupErr)
		}
		if ledger.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", ledger.createCalls)
		}
	})
}

func TestDepositWrapsCurrencyLookupError(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("currency lookup failed")
	accounts := &fakeAccountRepository{}
	currencies := &fakeCurrencyRepository{err: lookupErr}
	ledger := &fakeLedgerRepository{}
	service := testService(t, accounts, currencies, ledger)

	_, err := service.Deposit(context.Background(), testDepositParams(t))
	if !errors.Is(err, lookupErr) {
		t.Fatalf("Deposit() error = %v, want %v", err, lookupErr)
	}
	if ledger.createCalls != 0 {
		t.Fatalf("Create() calls = %d, want 0", ledger.createCalls)
	}
}

func TestDepositWrapsLedgerError(t *testing.T) {
	t.Parallel()

	createErr := errors.New("tigerbeetle unavailable")
	accounts := &fakeAccountRepository{}
	currencies := &fakeCurrencyRepository{}
	ledger := &fakeLedgerRepository{createErr: createErr}
	service := testService(t, accounts, currencies, ledger)

	_, err := service.Deposit(context.Background(), testDepositParams(t))
	if !errors.Is(err, createErr) {
		t.Fatalf("Deposit() error = %v, want %v", err, createErr)
	}
	if ledger.createCalls != 1 {
		t.Fatalf("Create() calls = %d, want 1", ledger.createCalls)
	}
}
