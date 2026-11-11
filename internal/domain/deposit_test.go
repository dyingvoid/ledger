package domain

import (
	"errors"
	"testing"
)

const testDepositCurrency Currency = "USD"

func TestNewDeposit(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid deposit", func(t *testing.T) {
		t.Parallel()

		deposit, err := NewDeposit(7, ExternalID{}, 2500, testDepositCurrency)
		if err != nil {
			t.Fatalf("NewDeposit() error = %v", err)
		}
		if deposit.AccountID != 7 {
			t.Fatalf("AccountID = %d, want 7", deposit.AccountID)
		}
		if deposit.AmountMinor != 2500 {
			t.Fatalf("AmountMinor = %d, want 2500", deposit.AmountMinor)
		}
		if deposit.Currency != testDepositCurrency {
			t.Fatalf("Currency = %q, want %q", deposit.Currency, "USD")
		}
	})

	t.Run("accepts external account reference", func(t *testing.T) {
		t.Parallel()

		externalAccountID, err := ParseExternalID("42")
		if err != nil {
			t.Fatalf("ParseExternalID() error = %v", err)
		}

		deposit, err := NewDeposit(0, externalAccountID, 2500, testDepositCurrency)
		if err != nil {
			t.Fatalf("NewDeposit() error = %v", err)
		}
		if deposit.ExternalAccountID != externalAccountID {
			t.Fatalf("ExternalAccountID = %s, want %s", deposit.ExternalAccountID, externalAccountID)
		}
	})

	tests := []struct {
		name        string
		accountID   int64
		amountMinor int64
		currency    Currency
		want        error
	}{
		{
			name:        "rejects missing account",
			accountID:   0,
			amountMinor: 2500,
			currency:    testDepositCurrency,
			want:        ErrDepositAccountRequired,
		},
		{
			name:        "rejects zero amount",
			accountID:   7,
			amountMinor: 0,
			currency:    testDepositCurrency,
			want:        ErrDepositAmountInvalid,
		},
		{
			name:        "rejects negative amount",
			accountID:   7,
			amountMinor: -2500,
			currency:    testDepositCurrency,
			want:        ErrDepositAmountInvalid,
		},
		{
			name:        "rejects empty currency",
			accountID:   7,
			amountMinor: 2500,
			currency:    "  ",
			want:        ErrDepositCurrencyRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewDeposit(test.accountID, ExternalID{}, test.amountMinor, test.currency)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewDeposit() error = %v, want %v", err, test.want)
			}
		})
	}
}
