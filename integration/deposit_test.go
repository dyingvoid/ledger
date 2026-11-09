//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres/repositories"
)

type accountBalance struct {
	DebitMinor  int64
	CreditMinor int64
}

func TestAddDepositCreatesDoubleEntry(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	clientAccount := createIntegrationAccount(t, ctx, 2100, "USD")
	bankAccountID := integrationBankAccountID(t, ctx, "USD")
	bankBefore := integrationBalance(t, ctx, bankAccountID)

	service := deposits.NewService(repositories.NewDeposits(testPool))
	deposit, err := service.AddDeposit(ctx, deposits.AddDepositParams{
		AccountID:   clientAccount.ID,
		AmountMinor: 2500,
		Currency:    "USD",
	})
	if err != nil {
		t.Fatalf("AddDeposit() error = %v", err)
	}
	if deposit.JournalEntryID <= 0 {
		t.Fatalf("JournalEntryID = %d, want a positive ID", deposit.JournalEntryID)
	}

	bankAfter := integrationBalance(t, ctx, bankAccountID)
	if bankAfter.DebitMinor != bankBefore.DebitMinor+2500 {
		t.Fatalf(
			"bank debit = %d, want %d",
			bankAfter.DebitMinor,
			bankBefore.DebitMinor+2500,
		)
	}

	clientAfter := integrationBalance(t, ctx, clientAccount.ID)
	if clientAfter.CreditMinor != 2500 {
		t.Fatalf("client credit = %d, want 2500", clientAfter.CreditMinor)
	}

	var entryType string
	if err := testPool.QueryRow(
		ctx,
		`SELECT entry_type FROM ledger.journal_entry WHERE id = $1`,
		deposit.JournalEntryID,
	).Scan(&entryType); err != nil {
		t.Fatalf("reading journal entry: %v", err)
	}
	if entryType != "deposit" {
		t.Fatalf("entry type = %q, want %q", entryType, "deposit")
	}

	var lineCount int
	if err := testPool.QueryRow(
		ctx,
		`SELECT count(*)
		 FROM ledger.journal_line
		 WHERE journal_entry_id = $1
		   AND amount_minor = $2
		   AND ((account_id = $3 AND side = 'DR')
		     OR (account_id = $4 AND side = 'CR'))`,
		deposit.JournalEntryID,
		2500,
		bankAccountID,
		clientAccount.ID,
	).Scan(&lineCount); err != nil {
		t.Fatalf("reading journal lines: %v", err)
	}
	if lineCount != 2 {
		t.Fatalf("matching journal lines = %d, want 2", lineCount)
	}
}

func TestAddDepositRejectsInvalidRequests(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	depositRepository := repositories.NewDeposits(testPool)
	clientAccount := createIntegrationAccount(t, ctx, 2100, "USD")
	foreignAccount := createIntegrationAccount(t, ctx, 2100, "CNY")
	bankAccountID := integrationBankAccountID(t, ctx, "USD")

	tests := []struct {
		name      string
		accountID int64
		amount    int64
		currency  domain.Currency
		want      error
	}{
		{
			name:      "unknown account",
			accountID: 9223372036854775807,
			amount:    2500,
			currency:  "USD",
			want:      domain.ErrDepositAccountNotFound,
		},
		{
			name:      "bank cash account",
			accountID: bankAccountID,
			amount:    2500,
			currency:  "USD",
			want:      domain.ErrDepositAccountNotClient,
		},
		{
			name:      "currency mismatch",
			accountID: foreignAccount.ID,
			amount:    2500,
			currency:  "USD",
			want:      domain.ErrDepositCurrencyMismatch,
		},
		{
			name:      "non-positive amount",
			accountID: clientAccount.ID,
			amount:    0,
			currency:  "USD",
			want:      domain.ErrDepositAmountInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := depositRepository.Create(ctx, domain.Deposit{
				AccountID:   test.accountID,
				AmountMinor: test.amount,
				Currency:    test.currency,
			})
			if !errors.Is(err, test.want) {
				t.Fatalf("Create() error = %v, want %v", err, test.want)
			}
		})
	}
}

func createIntegrationAccount(
	t *testing.T,
	ctx context.Context,
	accountType domain.AccountType,
	currency domain.Currency,
) domain.Account {
	t.Helper()

	account, err := domain.NewAccount(7, "Integration "+t.Name(), "", accountType, currency)
	if err != nil {
		t.Fatalf("NewAccount() error = %v", err)
	}

	created, err := repositories.NewAccounts(testPool).Create(ctx, account)
	if err != nil {
		t.Fatalf("creating account: %v", err)
	}
	return created
}

func integrationBankAccountID(t *testing.T, ctx context.Context, currency domain.Currency) int64 {
	t.Helper()

	var id int64
	if err := testPool.QueryRow(
		ctx,
		`SELECT id
		 FROM ledger.account
		 WHERE account_type = 1000
		   AND currency = $1
		   AND deleted_at IS NULL
		 ORDER BY id
		 LIMIT 1`,
		string(currency),
	).Scan(&id); err != nil {
		t.Fatalf("reading bank account: %v", err)
	}
	return id
}

func integrationBalance(t *testing.T, ctx context.Context, accountID int64) accountBalance {
	t.Helper()

	var balance accountBalance
	if err := testPool.QueryRow(
		ctx,
		`SELECT dr, cr FROM ledger.account WHERE id = $1`,
		accountID,
	).Scan(&balance.DebitMinor, &balance.CreditMinor); err != nil {
		t.Fatalf("reading account balance: %v", err)
	}
	return balance
}
