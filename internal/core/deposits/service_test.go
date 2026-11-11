package deposits

import (
	"context"
	"errors"
	"testing"

	"github.com/dyingvoid/ledger/internal/domain"
)

const testCurrency domain.Currency = "USD"

type fakeDepositRepository struct {
	createErr   error
	created     domain.Deposit
	createCalls int
}

func (f *fakeDepositRepository) Create(_ context.Context, deposit domain.Deposit) (domain.Deposit, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.Deposit{}, f.createErr
	}

	deposit.JournalEntryID = 42
	f.created = deposit

	return deposit, nil
}

func TestAddDeposit(t *testing.T) {
	t.Parallel()

	t.Run("adds a valid deposit", func(t *testing.T) {
		t.Parallel()

		repository := &fakeDepositRepository{}
		service := NewService(repository)

		deposit, err := service.AddDeposit(context.Background(), AddDepositParams{
			AccountID:   7,
			AmountMinor: 2500,
			Currency:    testCurrency,
		})
		if err != nil {
			t.Fatalf("AddDeposit() error = %v", err)
		}
		if deposit.JournalEntryID != 42 {
			t.Fatalf("JournalEntryID = %d, want 42", deposit.JournalEntryID)
		}
		if repository.createCalls != 1 {
			t.Fatalf("Create() calls = %d, want 1", repository.createCalls)
		}
		if repository.created.AmountMinor != 2500 {
			t.Fatalf("AmountMinor = %d, want 2500", repository.created.AmountMinor)
		}
	})

	t.Run("accepts an external account ID", func(t *testing.T) {
		t.Parallel()

		repository := &fakeDepositRepository{}
		service := NewService(repository)

		externalAccountID, err := domain.ParseExternalID("42")
		if err != nil {
			t.Fatalf("ParseExternalID() error = %v", err)
		}
		transferID, err := domain.ParseExternalID("99")
		if err != nil {
			t.Fatalf("ParseExternalID() error = %v", err)
		}

		deposit, err := service.AddDeposit(context.Background(), AddDepositParams{
			ExternalAccountID: externalAccountID,
			TransferID:        transferID,
			AmountMinor:       2500,
			Currency:          testCurrency,
		})
		if err != nil {
			t.Fatalf("AddDeposit() error = %v", err)
		}
		if deposit.ExternalAccountID != externalAccountID {
			t.Fatalf("ExternalAccountID = %s, want %s", deposit.ExternalAccountID, externalAccountID)
		}
		if deposit.TransferID != transferID {
			t.Fatalf("TransferID = %s, want %s", deposit.TransferID, transferID)
		}
	})

	t.Run("rejects invalid deposit before calling the repository", func(t *testing.T) {
		t.Parallel()

		repository := &fakeDepositRepository{}
		service := NewService(repository)

		_, err := service.AddDeposit(context.Background(), AddDepositParams{
			AccountID:   7,
			AmountMinor: 0,
			Currency:    testCurrency,
		})
		if !errors.Is(err, domain.ErrDepositAmountInvalid) {
			t.Fatalf("AddDeposit() error = %v, want %v", err, domain.ErrDepositAmountInvalid)
		}
		if repository.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", repository.createCalls)
		}
	})

	t.Run("wraps repository error", func(t *testing.T) {
		t.Parallel()

		saveErr := errors.New("database unavailable")
		repository := &fakeDepositRepository{createErr: saveErr}
		service := NewService(repository)

		_, err := service.AddDeposit(context.Background(), AddDepositParams{
			AccountID:   7,
			AmountMinor: 2500,
			Currency:    testCurrency,
		})
		if !errors.Is(err, saveErr) {
			t.Fatalf("AddDeposit() error = %v, want %v", err, saveErr)
		}
	})
}
