package domain

import (
	"errors"
	"testing"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
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

	t.Run("round-trips through bytes", func(t *testing.T) {
		t.Parallel()

		id, err := ParseID("123456789")
		if err != nil {
			t.Fatalf("ParseID() error = %v", err)
		}

		if got := IDFromBytes(id.Bytes()); got != id {
			t.Fatalf("IDFromBytes() = %s, want %s", got, id)
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
			if _, err := ParseID(value); !errors.Is(err, domain_errors.ErrIDInvalid) {
				t.Fatalf("ParseID(%q) error = %v, want %v", value, err, domain_errors.ErrIDInvalid)
			}
		}
	})
}

func TestIDBytesAreBigEndian(t *testing.T) {
	t.Parallel()

	id, err := ParseID("256")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	want := [idSize]byte{idSize - 2: 0x01, idSize - 1: 0x00}
	if id.Bytes() != want {
		t.Fatalf("Bytes() = %x, want %x", id.Bytes(), want)
	}
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
		if _, err := accountType.NormalBalance(); !errors.Is(err, domain_errors.ErrAccountTypeInvalid) {
			t.Fatalf("NormalBalance() error = %v, want %v", err, domain_errors.ErrAccountTypeInvalid)
		}
	}
}

func TestAccountTypeValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		accountType AccountType
		wantErr     bool
	}{
		{accountType: 999, wantErr: true},
		{accountType: 1000, wantErr: false},
		{accountType: 5999, wantErr: false},
		{accountType: 6000, wantErr: true},
	}

	for _, test := range tests {
		err := test.accountType.Validate()
		if test.wantErr && !errors.Is(err, domain_errors.ErrAccountTypeInvalid) {
			t.Fatalf("Validate() for %d error = %v, want %v", test.accountType, err, domain_errors.ErrAccountTypeInvalid)
		}
		if !test.wantErr && err != nil {
			t.Fatalf("Validate() for %d error = %v, want nil", test.accountType, err)
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
		if !errors.Is(err, domain_errors.ErrAccountIDRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, domain_errors.ErrAccountIDRequired)
		}
	})

	t.Run("rejects missing user", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(testID(t, "1"), 0, "Daily", "", 2100, "USD")
		if !errors.Is(err, domain_errors.ErrAccountUserRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, domain_errors.ErrAccountUserRequired)
		}
	})

	t.Run("rejects account type outside the nominal range", func(t *testing.T) {
		t.Parallel()

		for _, accountType := range []AccountType{999, 6000} {
			_, err := NewAccount(testID(t, "1"), 7, "Daily", "", accountType, "USD")
			if !errors.Is(err, domain_errors.ErrAccountTypeInvalid) {
				t.Fatalf("NewAccount(%d) error = %v, want %v", accountType, err, domain_errors.ErrAccountTypeInvalid)
			}
		}
	})

	t.Run("rejects empty currency", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(testID(t, "1"), 7, "Daily", "", 2100, "  ")
		if !errors.Is(err, domain_errors.ErrAccountCurrencyRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, domain_errors.ErrAccountCurrencyRequired)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		t.Parallel()

		_, err := NewAccount(testID(t, "1"), 7, "  ", "", 2100, "USD")
		if !errors.Is(err, domain_errors.ErrAccountNameRequired) {
			t.Fatalf("NewAccount() error = %v, want %v", err, domain_errors.ErrAccountNameRequired)
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
