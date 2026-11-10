package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrJournalEntryDateRequired = errors.New("journal entry date is required")
	ErrJournalEntryTooFewLines  = errors.New("journal entry requires at least two lines")
	ErrJournalEntryUnbalanced   = errors.New("journal entry is unbalanced")
	ErrJournalLineAmountInvalid = errors.New("journal line amount must be positive")
	ErrJournalLineSideInvalid   = errors.New("journal line side must be debit or credit")
)

type JournalEntry struct {
	ID    int64
	Date  time.Time
	Lines []JournalLine
}

type JournalLine struct {
	AccountID   int64
	Side        BalanceSide
	AmountMinor int64
}

func NewJournalEntry(date time.Time, lines []JournalLine) (*JournalEntry, error) {
	entry := &JournalEntry{
		Date:  date,
		Lines: append([]JournalLine(nil), lines...),
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	return entry, nil
}

func (e JournalEntry) Total(side BalanceSide) int64 {
	var total int64
	for _, line := range e.Lines {
		if line.Side == side {
			total += line.AmountMinor
		}
	}
	return total
}

func (e JournalEntry) IsBalanced() bool {
	return e.Total(Debit) == e.Total(Credit)
}

func (e JournalEntry) Validate() error {
	if e.Date.IsZero() {
		return ErrJournalEntryDateRequired
	}
	if len(e.Lines) < 2 {
		return ErrJournalEntryTooFewLines
	}
	for i, line := range e.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("validating journal line %d: %w", i, err)
		}
	}
	if !e.IsBalanced() {
		return fmt.Errorf(
			"%w: debits %d, credits %d",
			ErrJournalEntryUnbalanced,
			e.Total(Debit),
			e.Total(Credit),
		)
	}
	return nil
}

func (l JournalLine) Validate() error {
	if err := validateAccountID(l.AccountID); err != nil {
		return err
	}
	if l.Side != Debit && l.Side != Credit {
		return ErrJournalLineSideInvalid
	}
	if l.AmountMinor <= 0 {
		return ErrJournalLineAmountInvalid
	}
	return nil
}

func validateAccountID(id int64) error {
	if id >= int64(minAccountType) && id <= int64(maxAccountType) {
		return nil
	}
	return fmt.Errorf("%w: %d", ErrAccountTypeInvalid, id)
}
