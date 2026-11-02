package domain

import (
	"errors"
	"testing"
)

func TestAccountType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   int64
		want AccountType
	}{
		{name: "asset lower bound", id: 1000, want: AccountTypeAsset},
		{name: "asset upper bound", id: 1999, want: AccountTypeAsset},
		{name: "liability lower bound", id: 2000, want: AccountTypeLiability},
		{name: "liability upper bound", id: 2999, want: AccountTypeLiability},
		{name: "equity lower bound", id: 3000, want: AccountTypeEquity},
		{name: "equity upper bound", id: 3999, want: AccountTypeEquity},
		{name: "revenue lower bound", id: 4000, want: AccountTypeRevenue},
		{name: "revenue upper bound", id: 4999, want: AccountTypeRevenue},
		{name: "expense lower bound", id: 5000, want: AccountTypeExpense},
		{name: "expense upper bound", id: 5999, want: AccountTypeExpense},
		{name: "unassigned range", id: 6000, want: AccountTypeUnknown},
		{name: "below range", id: 999, want: AccountTypeUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			account := Account{ID: test.id}
			if got := account.Type(); got != test.want {
				t.Fatalf("Type() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestAccountNormalBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   int64
		want BalanceSide
	}{
		{id: 1000, want: Debit},
		{id: 2000, want: Credit},
		{id: 3000, want: Credit},
		{id: 4000, want: Credit},
		{id: 5000, want: Debit},
	}

	for _, test := range tests {
		account := Account{ID: test.id}
		balance, err := account.NormalBalance()
		if err != nil {
			t.Fatalf("NormalBalance() for %d error = %v", test.id, err)
		}
		if balance != test.want {
			t.Fatalf("NormalBalance() for %d = %d, want %d", test.id, balance, test.want)
		}
	}

	if _, err := (Account{ID: 6000}).NormalBalance(); !errors.Is(err, ErrAccountTypeUnknown) {
		t.Fatalf("NormalBalance() error = %v, want %v", err, ErrAccountTypeUnknown)
	}
}

func TestNewAccount(t *testing.T) {
	t.Parallel()

	t.Run("trims name and description", func(t *testing.T) {
		t.Parallel()

		account, err := NewAccount(1000, "  Cash  ", "  Petty cash  ")
		if err != nil {
			t.Fatalf("NewAccount() error = %v", err)
		}
		if account.Name != "Cash" {
			t.Fatalf("Name = %q, want %q", account.Name, "Cash")
		}
		if account.Description != "Petty cash" {
			t.Fatalf("Description = %q, want %q", account.Description, "Petty cash")
		}
	})

	t.Run("rejects unknown account ID", func(t *testing.T) {
		t.Parallel()

		for _, id := range []int64{999, 6000, 9999} {
			if _, err := NewAccount(id, "Cash", ""); !errors.Is(err, ErrAccountTypeUnknown) {
				t.Fatalf("NewAccount(%d) error = %v, want %v", id, err, ErrAccountTypeUnknown)
			}
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		t.Parallel()

		if _, err := NewAccount(1000, "  ", ""); !errors.Is(err, ErrAccountNameRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountNameRequired)
		}
	})
}
