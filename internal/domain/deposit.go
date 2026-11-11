package domain

import (
	"errors"
	"strings"
)

var (
	ErrDepositAccountRequired  = errors.New("deposit account ID is required")
	ErrDepositAmountInvalid    = errors.New("deposit amount must be positive")
	ErrDepositCurrencyRequired = errors.New("deposit currency is required")
	ErrDepositAccountNotFound  = errors.New("deposit account not found")
	ErrDepositAccountNotClient = errors.New("account is not a client deposit account")
	ErrDepositCurrencyMismatch = errors.New("deposit currency does not match account currency")
)

type Deposit struct {
	JournalEntryID    int64      `json:"journal_entry_id"`
	TransferID        ExternalID `json:"transfer_id"`
	AccountID         int64      `json:"account_id"`
	ExternalAccountID ExternalID `json:"external_account_id"`
	AmountMinor       int64      `json:"amount_minor"`
	Currency          Currency   `json:"currency"`
}

func NewDeposit(
	accountID int64,
	externalAccountID ExternalID,
	amountMinor int64,
	currency Currency,
) (Deposit, error) {
	deposit := Deposit{
		AccountID:         accountID,
		ExternalAccountID: externalAccountID,
		AmountMinor:       amountMinor,
		Currency:          currency,
	}
	if err := deposit.Validate(); err != nil {
		return Deposit{}, err
	}
	return deposit, nil
}

func (d Deposit) Validate() error {
	if d.AccountID <= 0 && d.ExternalAccountID.IsZero() {
		return ErrDepositAccountRequired
	}
	if d.AmountMinor <= 0 {
		return ErrDepositAmountInvalid
	}
	if strings.TrimSpace(string(d.Currency)) == "" {
		return ErrDepositCurrencyRequired
	}
	return nil
}
