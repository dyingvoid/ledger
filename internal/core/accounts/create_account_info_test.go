package accounts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const (
	testAccountName     = "Daily"
	testAccountCurrency = "USD"
)

type fakeInfoRepository struct {
	createErr   error
	getErr      error
	created     domain.Account
	account     domain.Account
	createCalls int
	getCalls    int
}

func (f *fakeInfoRepository) Create(_ context.Context, account domain.Account) (domain.Account, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.Account{}, f.createErr
	}

	account.CreatedAt = time.Now()
	f.created = account
	return account, nil
}

func (f *fakeInfoRepository) Get(_ context.Context, _ domain.ID) (domain.Account, error) {
	f.getCalls++
	if f.getErr != nil {
		return domain.Account{}, f.getErr
	}
	return f.account, nil
}

type fakeLedgerRepository struct {
	err     error
	account domain.Account
	calls   int
}

func (f *fakeLedgerRepository) Create(_ context.Context, account domain.Account) (domain.Account, error) {
	f.calls++
	if f.err != nil {
		return domain.Account{}, f.err
	}
	f.account = account
	return account, nil
}

func testAccountID(t *testing.T) domain.ID {
	t.Helper()

	id, err := domain.ParseID("42")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}
	return id
}

func newTestService(
	infos InfoRepository,
	ledger LedgerRepository,
) *Service {
	return NewService(infos, ledger)
}

func TestCreateAccountInfo(t *testing.T) {
	t.Parallel()

	t.Run("creates account info", func(t *testing.T) {
		t.Parallel()

		infos := &fakeInfoRepository{}
		service := newTestService(infos, &fakeLedgerRepository{})

		account, err := service.CreateAccountInfo(context.Background(), CreateAccountInfoParams{
			ID:          testAccountID(t),
			UserID:      7,
			Name:        testAccountName,
			Description: "Spending",
			Type:        2100,
			Currency:    testAccountCurrency,
		})
		if err != nil {
			t.Fatalf("CreateAccountInfo() error = %v", err)
		}
		if account.ID != testAccountID(t) {
			t.Fatalf("ID = %s, want %s", account.ID, testAccountID(t))
		}
		if infos.createCalls != 1 {
			t.Fatalf("Create() calls = %d, want 1", infos.createCalls)
		}
		if infos.created.Type != 2100 {
			t.Fatalf("Type = %d, want %d", infos.created.Type, 2100)
		}
	})

	t.Run("rejects invalid account before calling the repository", func(t *testing.T) {
		t.Parallel()

		infos := &fakeInfoRepository{}
		service := newTestService(infos, &fakeLedgerRepository{})

		_, err := service.CreateAccountInfo(context.Background(), CreateAccountInfoParams{
			ID:       testAccountID(t),
			UserID:   0,
			Name:     testAccountName,
			Type:     2100,
			Currency: testAccountCurrency,
		})
		if !errors.Is(err, domain_errors.ErrAccountUserRequired) {
			t.Fatalf("CreateAccountInfo() error = %v, want %v", err, domain_errors.ErrAccountUserRequired)
		}
		if infos.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", infos.createCalls)
		}
	})

	t.Run("wraps repository error", func(t *testing.T) {
		t.Parallel()

		createErr := errors.New("database unavailable")
		infos := &fakeInfoRepository{createErr: createErr}
		service := newTestService(infos, &fakeLedgerRepository{})

		_, err := service.CreateAccountInfo(context.Background(), CreateAccountInfoParams{
			ID:       testAccountID(t),
			UserID:   7,
			Name:     testAccountName,
			Type:     2100,
			Currency: testAccountCurrency,
		})
		if !errors.Is(err, createErr) {
			t.Fatalf("CreateAccountInfo() error = %v, want %v", err, createErr)
		}
	})
}

func TestOpenLedgerAccount(t *testing.T) {
	t.Parallel()

	t.Run("opens the ledger account", func(t *testing.T) {
		t.Parallel()

		account := domain.Account{
			ID:       testAccountID(t),
			UserID:   7,
			Name:     testAccountName,
			Type:     2100,
			Currency: testAccountCurrency,
		}
		infos := &fakeInfoRepository{account: account}
		ledger := &fakeLedgerRepository{}
		service := newTestService(infos, ledger)

		opened, err := service.OpenLedgerAccount(context.Background(), account.ID)
		if err != nil {
			t.Fatalf("OpenLedgerAccount() error = %v", err)
		}
		if opened.ID != account.ID {
			t.Fatalf("ID = %s, want %s", opened.ID, account.ID)
		}
		if infos.getCalls != 1 {
			t.Fatalf("Get() calls = %d, want 1", infos.getCalls)
		}
		if ledger.calls != 1 {
			t.Fatalf("Create() calls = %d, want 1", ledger.calls)
		}
	})

	t.Run("returns account info error without calling the ledger", func(t *testing.T) {
		t.Parallel()

		getErr := errors.New("database unavailable")
		infos := &fakeInfoRepository{getErr: getErr}
		ledger := &fakeLedgerRepository{}
		service := newTestService(infos, ledger)

		_, err := service.OpenLedgerAccount(context.Background(), testAccountID(t))
		if !errors.Is(err, getErr) {
			t.Fatalf("OpenLedgerAccount() error = %v, want %v", err, getErr)
		}
		if ledger.calls != 0 {
			t.Fatalf("Create() calls = %d, want 0", ledger.calls)
		}
	})

	t.Run("wraps ledger error", func(t *testing.T) {
		t.Parallel()

		ledgerErr := errors.New("tigerbeetle unavailable")
		infos := &fakeInfoRepository{account: domain.Account{ID: testAccountID(t)}}
		ledger := &fakeLedgerRepository{err: ledgerErr}
		service := newTestService(infos, ledger)

		_, err := service.OpenLedgerAccount(context.Background(), testAccountID(t))
		if !errors.Is(err, ledgerErr) {
			t.Fatalf("OpenLedgerAccount() error = %v, want %v", err, ledgerErr)
		}
	})
}
