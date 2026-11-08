package domain

import (
	"errors"
	"testing"
	"time"
)

func testDate() time.Time {
	return time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
}

func TestNewJournalEntry(t *testing.T) {
	t.Parallel()

	t.Run("accepts balanced entry", func(t *testing.T) {
		t.Parallel()

		entry, err := NewJournalEntry(testDate(), []JournalLine{
			{AccountID: 1000, Side: Debit, AmountMinor: 2500},
			{AccountID: 4000, Side: Credit, AmountMinor: 2500},
		})
		if err != nil {
			t.Fatalf("NewJournalEntry() error = %v", err)
		}
		if !entry.IsBalanced() {
			t.Fatal("IsBalanced() = false, want true")
		}
	})

	t.Run("rejects unbalanced entry", func(t *testing.T) {
		t.Parallel()

		_, err := NewJournalEntry(testDate(), []JournalLine{
			{AccountID: 1000, Side: Debit, AmountMinor: 2500},
			{AccountID: 4000, Side: Credit, AmountMinor: 2000},
		})
		if !errors.Is(err, ErrJournalEntryUnbalanced) {
			t.Fatalf("NewJournalEntry() error = %v, want %v", err, ErrJournalEntryUnbalanced)
		}
	})

	t.Run("rejects single line", func(t *testing.T) {
		t.Parallel()

		_, err := NewJournalEntry(testDate(), []JournalLine{
			{AccountID: 1000, Side: Debit, AmountMinor: 2500},
		})
		if !errors.Is(err, ErrJournalEntryTooFewLines) {
			t.Fatalf("NewJournalEntry() error = %v, want %v", err, ErrJournalEntryTooFewLines)
		}
	})

	t.Run("rejects zero date", func(t *testing.T) {
		t.Parallel()

		_, err := NewJournalEntry(time.Time{}, []JournalLine{
			{AccountID: 1000, Side: Debit, AmountMinor: 2500},
			{AccountID: 4000, Side: Credit, AmountMinor: 2500},
		})
		if !errors.Is(err, ErrJournalEntryDateRequired) {
			t.Fatalf("NewJournalEntry() error = %v, want %v", err, ErrJournalEntryDateRequired)
		}
	})

	t.Run("rejects invalid line", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			line JournalLine
			want error
		}{
			{
				name: "unknown account",
				line: JournalLine{AccountID: 6000, Side: Debit, AmountMinor: 2500},
				want: ErrAccountTypeInvalid,
			},
			{
				name: "unset side",
				line: JournalLine{AccountID: 1000, AmountMinor: 2500},
				want: ErrJournalLineSideInvalid,
			},
			{
				name: "unknown side",
				line: JournalLine{AccountID: 1000, Side: BalanceSide(9), AmountMinor: 2500},
				want: ErrJournalLineSideInvalid,
			},
			{
				name: "zero amount",
				line: JournalLine{AccountID: 1000, Side: Debit, AmountMinor: 0},
				want: ErrJournalLineAmountInvalid,
			},
			{
				name: "negative amount",
				line: JournalLine{AccountID: 1000, Side: Debit, AmountMinor: -2500},
				want: ErrJournalLineAmountInvalid,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				_, err := NewJournalEntry(testDate(), []JournalLine{
					test.line,
					{AccountID: 4000, Side: Credit, AmountMinor: 2500},
				})
				if !errors.Is(err, test.want) {
					t.Fatalf("NewJournalEntry() error = %v, want %v", err, test.want)
				}
			})
		}
	})
}

func TestJournalEntryTotal(t *testing.T) {
	t.Parallel()

	entry := JournalEntry{Lines: []JournalLine{
		{AccountID: 1000, Side: Debit, AmountMinor: 1000},
		{AccountID: 1100, Side: Debit, AmountMinor: 250},
		{AccountID: 4000, Side: Credit, AmountMinor: 1250},
	}}

	if got := entry.Total(Debit); got != 1250 {
		t.Fatalf("Total(debit) = %d, want 1250", got)
	}
	if got := entry.Total(Credit); got != 1250 {
		t.Fatalf("Total(credit) = %d, want 1250", got)
	}
	if !entry.IsBalanced() {
		t.Fatal("IsBalanced() = false, want true")
	}
}
