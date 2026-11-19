package repositories

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/domain"
)

func testAccount(t *testing.T) domain.Account {
	t.Helper()

	id, err := domain.ParseID("42")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	return domain.Account{
		ID:          id,
		UserID:      7,
		Name:        "Daily",
		Description: "Spending",
		Type:        2100,
		Currency:    "USD",
	}
}

func TestSameAccountInfo(t *testing.T) {
	t.Parallel()

	account := testAccount(t)

	differentID, err := domain.ParseID("43")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	tests := []struct {
		name     string
		existing domain.Account
		want     bool
	}{
		{name: "identical", existing: account, want: true},
		{
			name:     "different ID",
			existing: withField(account, func(a *domain.Account) { a.ID = differentID }),
			want:     false,
		},
		{
			name:     "different user",
			existing: withField(account, func(a *domain.Account) { a.UserID = 8 }),
			want:     false,
		},
		{
			name:     "different name",
			existing: withField(account, func(a *domain.Account) { a.Name = "Other" }),
			want:     false,
		},
		{
			name:     "different description",
			existing: withField(account, func(a *domain.Account) { a.Description = "Other" }),
			want:     false,
		},
		{
			name:     "different type",
			existing: withField(account, func(a *domain.Account) { a.Type = 1000 }),
			want:     false,
		},
		{
			name:     "different currency",
			existing: withField(account, func(a *domain.Account) { a.Currency = "CNY" }),
			want:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := sameAccountInfo(test.existing, account); got != test.want {
				t.Fatalf("sameAccountInfo() = %t, want %t", got, test.want)
			}
		})
	}
}

func withField(account domain.Account, mutate func(*domain.Account)) domain.Account {
	mutate(&account)
	return account
}

func TestAccountError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		constraint string
		want       error
	}{
		{name: "account type", constraint: accountTypeForeignKey, want: accounts.ErrAccountTypeNotFound},
		{name: "currency", constraint: accountCurrencyForeignKey, want: accounts.ErrCurrencyNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := accountError(&pgconn.PgError{ConstraintName: test.constraint})
			if !errors.Is(err, test.want) {
				t.Fatalf("accountError() = %v, want %v", err, test.want)
			}
		})
	}

	if err := accountError(errors.New("boom")); err == nil {
		t.Fatal("accountError() = nil, want wrapped error")
	}
}
