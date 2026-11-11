package accounts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dyingvoid/ledger/internal/domain"
)

const (
	testAccountName     = "Daily"
	testAccountCurrency = "USD"
)

type fakeAccountRepository struct {
	createErr   error
	created     domain.Account
	createCalls int
}

func (f *fakeAccountRepository) Create(_ context.Context, account domain.Account) (domain.Account, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.Account{}, f.createErr
	}

	account.ID = 42
	account.CreatedAt = time.Now()
	f.created = account

	return account, nil
}

type fakeAccountTypeRepository struct {
	exists bool
	err    error
}

func (f *fakeAccountTypeRepository) AccountTypeExists(_ context.Context, _ domain.AccountType) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.exists, nil
}

type fakeCurrencyRepository struct {
	exists bool
	err    error
}

func (f *fakeCurrencyRepository) Exists(_ context.Context, _ domain.Currency) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.exists, nil
}

func newTestService(
	accounts AccountRepository,
	accountTypes AccountTypeRepository,
	currencies CurrencyRepository,
) *Service {
	return NewService(accounts, accountTypes, currencies)
}

//nolint:gocyclo,funlen // The validation subtests are clearer together.
func TestCreateAccount(t *testing.T) {
	t.Parallel()

	t.Run("creates a valid account", func(t *testing.T) {
		t.Parallel()

		repository := &fakeAccountRepository{}
		service := newTestService(
			repository,
			&fakeAccountTypeRepository{exists: true},
			&fakeCurrencyRepository{exists: true},
		)

		externalID, err := domain.ParseExternalID("42")
		if err != nil {
			t.Fatalf("ParseExternalID() error = %v", err)
		}

		account, err := service.CreateAccount(context.Background(), CreateAccountParams{
			UserID:      7,
			Name:        testAccountName,
			Description: "Spending",
			Type:        2100,
			Currency:    testAccountCurrency,
			ExternalID:  externalID,
		})
		if err != nil {
			t.Fatalf("CreateAccount() error = %v", err)
		}
		if account.ID != 42 {
			t.Fatalf("ID = %d, want 42", account.ID)
		}
		if repository.createCalls != 1 {
			t.Fatalf("Create() calls = %d, want 1", repository.createCalls)
		}
		if repository.created.Type != 2100 {
			t.Fatalf("Type = %d, want %d", repository.created.Type, 2100)
		}
		if repository.created.ExternalID != externalID {
			t.Fatalf("ExternalID = %s, want %s", repository.created.ExternalID, externalID)
		}
	})

	t.Run("rejects invalid account before checking the database", func(t *testing.T) {
		t.Parallel()

		repository := &fakeAccountRepository{}
		service := newTestService(
			repository,
			&fakeAccountTypeRepository{exists: true},
			&fakeCurrencyRepository{exists: true},
		)

		_, err := service.CreateAccount(context.Background(), CreateAccountParams{
			UserID:   0,
			Name:     testAccountName,
			Type:     2100,
			Currency: testAccountCurrency,
		})
		if !errors.Is(err, domain.ErrAccountUserRequired) {
			t.Fatalf("CreateAccount() error = %v, want %v", err, domain.ErrAccountUserRequired)
		}
		if repository.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", repository.createCalls)
		}
	})

	t.Run("rejects unknown account type", func(t *testing.T) {
		t.Parallel()

		repository := &fakeAccountRepository{}
		service := newTestService(
			repository,
			&fakeAccountTypeRepository{exists: false},
			&fakeCurrencyRepository{exists: true},
		)

		_, err := service.CreateAccount(context.Background(), CreateAccountParams{
			UserID:   7,
			Name:     testAccountName,
			Type:     2100,
			Currency: testAccountCurrency,
		})
		if !errors.Is(err, ErrAccountTypeNotFound) {
			t.Fatalf("CreateAccount() error = %v, want %v", err, ErrAccountTypeNotFound)
		}
		if repository.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", repository.createCalls)
		}
	})

	t.Run("rejects unknown currency", func(t *testing.T) {
		t.Parallel()

		repository := &fakeAccountRepository{}
		service := newTestService(
			repository,
			&fakeAccountTypeRepository{exists: true},
			&fakeCurrencyRepository{exists: false},
		)

		_, err := service.CreateAccount(context.Background(), CreateAccountParams{
			UserID:   7,
			Name:     testAccountName,
			Type:     2100,
			Currency: "EUR",
		})
		if !errors.Is(err, ErrCurrencyNotFound) {
			t.Fatalf("CreateAccount() error = %v, want %v", err, ErrCurrencyNotFound)
		}
		if repository.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", repository.createCalls)
		}
	})

	t.Run("wraps repository error", func(t *testing.T) {
		t.Parallel()

		saveErr := errors.New("database unavailable")
		repository := &fakeAccountRepository{createErr: saveErr}
		service := newTestService(
			repository,
			&fakeAccountTypeRepository{exists: true},
			&fakeCurrencyRepository{exists: true},
		)

		_, err := service.CreateAccount(context.Background(), CreateAccountParams{
			UserID:   7,
			Name:     testAccountName,
			Type:     2100,
			Currency: testAccountCurrency,
		})
		if !errors.Is(err, saveErr) {
			t.Fatalf("CreateAccount() error = %v, want %v", err, saveErr)
		}
	})
}
