package domain

import (
	"strings"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

type Deposit struct {
	ID          ID     `json:"id"`
	AccountID   ID     `json:"account_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func NewDeposit(id ID, accountID ID, amountMinor int64, currency string) (Deposit, error) {
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
	if strings.TrimSpace(d.Currency) == "" {
		return domain_errors.ErrDepositCurrencyRequired
	}
	return nil
}
