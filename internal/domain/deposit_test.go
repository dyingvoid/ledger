package domain

import (
	"errors"
	"testing"
	"time"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const testDepositCurrency = "USD"

func testDepositCurrencyInfo() CurrencyInfo {
	return CurrencyInfo{
		Code:       testDepositCurrency,
		Name:       "US Dollar",
		Ledger:     1,
		AssetScale: 2,
	}
}

func testDepositAccount(t *testing.T, id string, accountType AccountType, currency string) Account {
	t.Helper()

	return Account{
		ID:       testID(t, id),
		UserID:   1,
		Name:     "Account " + id,
		Type:     accountType,
		Currency: currency,
	}
}

func testValidDeposit(t *testing.T) Deposit {
	t.Helper()

	deposit, err := NewDeposit(
		testID(t, "9"),
		2500,
		testDepositCurrencyInfo(),
		testDepositAccount(t, "1", AccountTypeClientDeposit, testDepositCurrency),
		testDepositAccount(t, "2", AccountTypeBankCash, testDepositCurrency),
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}
	return deposit
}

func TestNewDeposit(t *testing.T) {
	t.Parallel()

	t.Run("accepts a valid deposit", func(t *testing.T) {
		t.Parallel()

		deposit := testValidDeposit(t)
		if deposit.ID.String() != "9" {
			t.Fatalf("ID = %s, want 9", deposit.ID)
		}
		if deposit.ClientAccount.ID.String() != "1" {
			t.Fatalf("ClientAccount.ID = %s, want 1", deposit.ClientAccount.ID)
		}
		if deposit.BankAccount.ID.String() != "2" {
			t.Fatalf("BankAccount.ID = %s, want 2", deposit.BankAccount.ID)
		}
		if deposit.AmountMinor != 2500 {
			t.Fatalf("AmountMinor = %d, want 2500", deposit.AmountMinor)
		}
		if deposit.Currency.Code != testDepositCurrency {
			t.Fatalf("Currency.Code = %q, want %q", deposit.Currency.Code, testDepositCurrency)
		}
	})

	tests := []struct {
		name   string
		mutate func(*Deposit)
		want   error
	}{
		{
			name:   "rejects missing deposit ID",
			mutate: func(d *Deposit) { d.ID = ID{} },
			want:   domain_errors.ErrDepositIDRequired,
		},
		{
			name:   "rejects zero amount",
			mutate: func(d *Deposit) { d.AmountMinor = 0 },
			want:   domain_errors.ErrDepositAmountInvalid,
		},
		{
			name:   "rejects negative amount",
			mutate: func(d *Deposit) { d.AmountMinor = -1 },
			want:   domain_errors.ErrDepositAmountInvalid,
		},
		{
			name:   "rejects empty currency code",
			mutate: func(d *Deposit) { d.Currency.Code = " " },
			want:   domain_errors.ErrCurrencyCodeRequired,
		},
		{
			name:   "rejects missing client account",
			mutate: func(d *Deposit) { d.ClientAccount.ID = ID{} },
			want:   domain_errors.ErrDepositClientAccountRequired,
		},
		{
			name:   "rejects missing bank account",
			mutate: func(d *Deposit) { d.BankAccount.ID = ID{} },
			want:   domain_errors.ErrDepositBankAccountRequired,
		},
		{
			name:   "rejects identical accounts",
			mutate: func(d *Deposit) { d.BankAccount.ID = d.ClientAccount.ID },
			want:   domain_errors.ErrDepositAccountsIdentical,
		},
		{
			name: "rejects deleted client account",
			mutate: func(d *Deposit) {
				deleted := time.Now()
				d.ClientAccount.DeletedAt = &deleted
			},
			want: domain_errors.ErrDepositAccountDeleted,
		},
		{
			name:   "rejects client account that does not support deposits",
			mutate: func(d *Deposit) { d.ClientAccount.Type = AccountTypeBankCash },
			want:   domain_errors.ErrDepositClientAccountTypeUnsupported,
		},
		{
			name:   "rejects bank account that cannot fund deposits",
			mutate: func(d *Deposit) { d.BankAccount.Type = AccountTypeClientDeposit },
			want:   domain_errors.ErrDepositBankAccountTypeUnsupported,
		},
		{
			name:   "rejects client currency mismatch",
			mutate: func(d *Deposit) { d.ClientAccount.Currency = "CNY" },
			want:   domain_errors.ErrDepositClientCurrencyMismatch,
		},
		{
			name:   "rejects bank currency mismatch",
			mutate: func(d *Deposit) { d.BankAccount.Currency = "CNY" },
			want:   domain_errors.ErrDepositBankCurrencyMismatch,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deposit := testValidDeposit(t)
			test.mutate(&deposit)

			if err := deposit.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}
