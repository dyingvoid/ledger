package domain

import (
	"strings"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
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
		return domain_errors.ErrDepositIDRequired
	}
	if d.AccountID.IsZero() {
		return domain_errors.ErrDepositAccountRequired
	}
	if d.AmountMinor <= 0 {
		return domain_errors.ErrDepositAmountInvalid
	}
	if strings.TrimSpace(string(d.Currency)) == "" {
		return domain_errors.ErrDepositCurrencyRequired
	}
	return nil
}
