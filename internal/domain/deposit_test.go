package domain

import (
	"errors"
	"testing"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const testDepositCurrency Currency = "USD"

func TestNewDeposit(t *testing.T) {
	t.Parallel()

	t.Run("accepts a valid deposit", func(t *testing.T) {
		t.Parallel()

		deposit, err := NewDeposit(testID(t, "9"), testID(t, "1"), 2500, testDepositCurrency)
		if err != nil {
			t.Fatalf("NewDeposit() error = %v", err)
		}
		if deposit.ID.String() != "9" {
			t.Fatalf("ID = %s, want 9", deposit.ID)
		}
		if deposit.AccountID.String() != "1" {
			t.Fatalf("AccountID = %s, want 1", deposit.AccountID)
		}
		if deposit.AmountMinor != 2500 {
			t.Fatalf("AmountMinor = %d, want 2500", deposit.AmountMinor)
		}
		if deposit.Currency != testDepositCurrency {
			t.Fatalf("Currency = %q, want %q", deposit.Currency, testDepositCurrency)
		}
	})

	tests := []struct {
		name      string
		id        ID
		accountID ID
		amount    int64
		currency  Currency
		want      error
	}{
		{
			name:      "rejects missing deposit ID",
			id:        ID{},
			accountID: testID(t, "1"),
			amount:    2500,
			currency:  testDepositCurrency,
			want:      domain_errors.ErrDepositIDRequired,
		},
		{
			name:      "rejects missing account",
			id:        testID(t, "9"),
			accountID: ID{},
			amount:    2500,
			currency:  testDepositCurrency,
			want:      domain_errors.ErrDepositAccountRequired,
		},
		{
			name:      "rejects zero amount",
			id:        testID(t, "9"),
			accountID: testID(t, "1"),
			amount:    0,
			currency:  testDepositCurrency,
			want:      domain_errors.ErrDepositAmountInvalid,
		},
		{
			name:      "rejects negative amount",
			id:        testID(t, "9"),
			accountID: testID(t, "1"),
			amount:    -2500,
			currency:  testDepositCurrency,
			want:      domain_errors.ErrDepositAmountInvalid,
		},
		{
			name:      "rejects empty currency",
			id:        testID(t, "9"),
			accountID: testID(t, "1"),
			amount:    2500,
			currency:  "  ",
			want:      domain_errors.ErrDepositCurrencyRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewDeposit(test.id, test.accountID, test.amount, test.currency)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewDeposit() error = %v, want %v", err, test.want)
			}
		})
	}
}
