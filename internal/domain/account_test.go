package domain

import (
	"errors"
	"testing"
)

func TestParseID(t *testing.T) {
	t.Parallel()

	t.Run("accepts decimal and hexadecimal values", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			value string
			want  string
		}{
			{value: "1", want: "1"},
			{value: "42", want: "42"},
			{value: "0xff", want: "255"},
			{value: "340282366920938463463374607431768211454", want: "340282366920938463463374607431768211454"},
		}

		for _, test := range tests {
			id, err := ParseID(test.value)
			if err != nil {
				t.Fatalf("ParseID(%q) error = %v", test.value, err)
			}
			if id.String() != test.want {
				t.Fatalf("ParseID(%q) = %s, want %s", test.value, id, test.want)
			}
		}
	})

	t.Run("round-trips through big-endian storage", func(t *testing.T) {
		t.Parallel()

		id, err := ParseID("123456789")
		if err != nil {
			t.Fatalf("ParseID() error = %v", err)
		}

		stored := id.BigEndianBytes()
		if got := IDFromBigEndianBytes(stored); got != id {
			t.Fatalf("IDFromBigEndianBytes() = %s, want %s", got, id)
		}
		if bytes := id.Bytes(); IDFromBytes(bytes) != id {
			t.Fatalf("IDFromBytes() = %s, want %s", IDFromBytes(bytes), id)
		}
	})

	t.Run("rejects invalid values", func(t *testing.T) {
		t.Parallel()

		tests := []string{
			"",
			"   ",
			"abc",
			"-1",
			"0",
			"0x0",
			"340282366920938463463374607431768211456",
			"340282366920938463463374607431768211455",
		}

		for _, value := range tests {
			if _, err := ParseID(value); !errors.Is(err, ErrIDInvalid) {
				t.Fatalf("ParseID(%q) error = %v, want %v", value, err, ErrIDInvalid)
			}
		}
	})
}

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

		account, err := NewAccount(testID(t, "1"), 7, "  Daily  ", "  Spending  ", 2100, "USD")
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

	t.Run("rejects missing ID", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(ID{}, 7, "Daily", "", 2100, "USD")
		if !errors.Is(err, ErrAccountIDRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountIDRequired)
		}
	})

	t.Run("rejects missing user", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(testID(t, "1"), 0, "Daily", "", 2100, "USD")
		if !errors.Is(err, ErrAccountUserRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountUserRequired)
		}
	})

	t.Run("rejects account type outside the nominal range", func(t *testing.T) {
		t.Parallel()

		for _, accountType := range []AccountType{999, 6000} {
			_, err := NewAccount(testID(t, "1"), 7, "Daily", "", accountType, "USD")
			if !errors.Is(err, ErrAccountTypeInvalid) {
				t.Fatalf("NewAccount(%d) error = %v, want %v", accountType, err, ErrAccountTypeInvalid)
			}
		}
	})

	t.Run("rejects empty currency", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(testID(t, "1"), 7, "Daily", "", 2100, "  ")
		if !errors.Is(err, ErrAccountCurrencyRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountCurrencyRequired)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(testID(t, "1"), 7, "  ", "", 2100, "USD")
		if !errors.Is(err, ErrAccountNameRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, ErrAccountNameRequired)
		}
	})
}

func testID(t *testing.T, value string) ID {
	t.Helper()

	id, err := ParseID(value)
	if err != nil {
		t.Fatalf("ParseID(%q) error = %v", value, err)
	}
	return id
}
