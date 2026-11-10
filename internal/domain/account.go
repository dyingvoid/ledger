package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrAccountNameRequired = errors.New("account name is required")
	ErrAccountUserRequired = errors.New("account user ID is required")
)

type Account struct {
	ID          int64       `json:"id"`
	ExternalID  ExternalID  `json:"external_id"`
	UserID      int64       `json:"user_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        AccountType `json:"type"`
	Currency    Currency    `json:"currency"`
	DebitMinor  int64       `json:"debit_minor"`
	CreditMinor int64       `json:"credit_minor"`
	DeletedAt   *time.Time  `json:"deleted_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

func NewAccount(
	userID int64,
	name string,
	description string,
	accountType AccountType,
	currency Currency,
) (Account, error) {
	account := Account{
		UserID:      userID,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Type:        accountType,
		Currency:    currency,
	}
	if err := account.Validate(); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (a Account) NormalBalance() (BalanceSide, error) {
	return a.Type.NormalBalance()
}

func (a Account) Validate() error {
	if a.UserID <= 0 {
		return ErrAccountUserRequired
	}
	if strings.TrimSpace(a.Name) == "" {
		return ErrAccountNameRequired
	}
	if err := a.Type.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(string(a.Currency)) == "" {
		return ErrAccountCurrencyRequired
	}
	return nil
}
