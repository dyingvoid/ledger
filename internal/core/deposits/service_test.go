package deposits

import (
	"context"
	"errors"
	"testing"

	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const testCurrency = "USD"

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

func TestDeposit(t *testing.T) {
	t.Parallel()

	t.Run("creates a valid deposit", func(t *testing.T) {
		t.Parallel()

		repository := &fakeDepositRepository{}
		service := NewService(repository)

		deposit, err := service.Deposit(context.Background(), DepositParams{
			ID:          testID(t, "9"),
			AccountID:   testID(t, "1"),
			AmountMinor: 2500,
			Currency:    testCurrency,
		})
		if err != nil {
			t.Fatalf("Deposit() error = %v", err)
		}
		if deposit.ID.String() != "9" {
			t.Fatalf("ID = %s, want 9", deposit.ID)
		}
		if repository.createCalls != 1 {
			t.Fatalf("Create() calls = %d, want 1", repository.createCalls)
		}
		if repository.created.AmountMinor != 2500 {
			t.Fatalf("AmountMinor = %d, want 2500", repository.created.AmountMinor)
		}
	})

	t.Run("rejects invalid deposit before calling the repository", func(t *testing.T) {
		t.Parallel()

		repository := &fakeDepositRepository{}
		service := NewService(repository)

		_, err := service.Deposit(context.Background(), DepositParams{
			ID:          testID(t, "9"),
			AccountID:   testID(t, "1"),
			AmountMinor: 0,
			Currency:    testCurrency,
		})
		if !errors.Is(err, domain_errors.ErrDepositAmountInvalid) {
			t.Fatalf("Deposit() error = %v, want %v", err, domain_errors.ErrDepositAmountInvalid)
		}
		if repository.createCalls != 0 {
			t.Fatalf("Create() calls = %d, want 0", repository.createCalls)
		}
	})

	t.Run("wraps repository error", func(t *testing.T) {
		t.Parallel()

		createErr := errors.New("tigerbeetle unavailable")
		repository := &fakeDepositRepository{createErr: createErr}
		service := NewService(repository)

		_, err := service.Deposit(context.Background(), DepositParams{
			ID:          testID(t, "9"),
			AccountID:   testID(t, "1"),
			AmountMinor: 2500,
			Currency:    testCurrency,
		})
		if !errors.Is(err, createErr) {
			t.Fatalf("Deposit() error = %v, want %v", err, createErr)
		}
	})
}
