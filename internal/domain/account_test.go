package domain

import (
	"errors"
	"testing"
)

func TestAccountTypeNormalBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		accountType AccountType
		want        BalanceSide
	}{
		{name: "asset lower bound", accountType: 1000, want: Debit},
		{name: "asset upper bound", accountType: 1999, want: Debit},
		{name: "liability lower bound", accountType: 2000, want: Credit},
		{name: "revenue upper bound", accountType: 4999, want: Credit},
		{name: "expense lower bound", accountType: 5000, want: Debit},
		{name: "expense upper bound", accountType: 5999, want: Debit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			balance, err := test.accountType.NormalBalance()
			if err != nil {
				t.Fatalf("NormalBalance() error = %v", err)
			}
			if balance != test.want {
				t.Fatalf("NormalBalance() = %d, want %d", balance, test.want)
			}
		})
	}

	for _, accountType := range []AccountType{999, 6000} {
		if _, err := accountType.NormalBalance(); !errors.Is(err, ErrAccountTypeInvalid) {
			t.Fatalf("NormalBalance() error = %v, want %v", err, ErrAccountTypeInvalid)
		}
	}
}

func TestAccountTypeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		accountType AccountType
		want        bool
	}{
		{accountType: 999, want: false},
		{accountType: 1000, want: true},
		{accountType: 5999, want: true},
		{accountType: 6000, want: false},
	}

	for _, test := range tests {
		if got := test.accountType.Valid(); got != test.want {
			t.Fatalf("Valid() for %d = %t, want %t", test.accountType, got, test.want)
		}
	}
}

func TestNewAccount(t *testing.T) {
	t.Parallel()

	t.Run("trims name and description", func(t *testing.T) {
		t.Parallel()

		account, err := NewAccount(7, "  Daily  ", "  Spending  ", 2100, "USD")
		if err != nil {
			t.Fatalf("NewAccount() error = %v", err)
		}
		if account.Name != "Daily" {
			t.Fatalf("Name = %q, want %q", account.Name, "Daily")
		}
		if account.Description != "Spending" {
			t.Fatalf("Description = %q, want %q", account.Description, "Spending")
		}
	})

	t.Run("rejects missing user", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(0, "Daily", "", 2100, "USD")
		if !errors.Is(err, ErrAccountUserRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountUserRequired)
		}
	})

	t.Run("rejects account type outside the nominal range", func(t *testing.T) {
		t.Parallel()

		for _, accountType := range []AccountType{999, 6000} {
			_, err := NewAccount(7, "Daily", "", accountType, "USD")
			if !errors.Is(err, ErrAccountTypeInvalid) {
				t.Fatalf("NewAccount(%d) error = %v, want %v", accountType, err, ErrAccountTypeInvalid)
			}
		}
	})

	t.Run("rejects empty currency", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(7, "Daily", "", 2100, "  ")
		if !errors.Is(err, ErrAccountCurrencyRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountCurrencyRequired)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(7, "  ", "", 2100, "USD")
		if !errors.Is(err, ErrAccountNameRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountNameRequired)
		}
	})
}
