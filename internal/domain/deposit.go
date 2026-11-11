package domain

import (
	"errors"
	"strings"
)

var (
	ErrDepositIDRequired       = errors.New("deposit ID is required")
	ErrDepositAccountRequired  = errors.New("deposit account ID is required")
	ErrDepositAmountInvalid    = errors.New("deposit amount must be positive")
	ErrDepositCurrencyRequired = errors.New("deposit currency is required")
	ErrDepositAccountNotFound  = errors.New("deposit account not found")
	ErrDepositAccountNotClient = errors.New("account is not a client deposit account")
	ErrDepositCurrencyMismatch = errors.New("deposit currency does not match account currency")
	ErrDepositConflict         = errors.New("deposit conflicts with an existing transfer")
)

// Deposit is a request to move funds into a client account. Its ID is the
// TigerBeetle transfer ID and therefore also its idempotency key.
type Deposit struct {
	ID          ID       `json:"id"`
	AccountID   ID       `json:"account_id"`
	AmountMinor int64    `json:"amount_minor"`
	Currency    Currency `json:"currency"`
}

func NewDeposit(id ID, accountID ID, amountMinor int64, currency Currency) (Deposit, error) {
	deposit := Deposit{
		ID:          id,
		AccountID:   accountID,
		AmountMinor: amountMinor,
		Currency:    currency,
	}
	if err := deposit.Validate(); err != nil {
		return Deposit{}, err
	}
	return deposit, nil
}

func (d Deposit) Validate() error {
	if d.ID.IsZero() {
		return ErrDepositIDRequired
	}
	if d.AccountID.IsZero() {
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
