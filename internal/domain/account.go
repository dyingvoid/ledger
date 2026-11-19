package domain

import (
	"strings"
	"time"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

type Account struct {
	ID          ID          `json:"id"`
	UserID      int64       `json:"user_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        AccountType `json:"type"`
	Currency    string      `json:"currency"`
	DeletedAt   *time.Time  `json:"deleted_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

func NewAccount(
	id ID,
	userID int64,
	name string,
	description string,
	accountType AccountType,
	currency string,
) (Account, error) {
	account := Account{
		ID:          id,
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
	if a.ID.IsZero() {
		return domain_errors.ErrAccountIDRequired
	}
	if a.UserID <= 0 {
		return domain_errors.ErrAccountUserRequired
	}
	if strings.TrimSpace(a.Name) == "" {
		return domain_errors.ErrAccountNameRequired
	}
	if err := a.Type.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(a.Currency) == "" {
		return domain_errors.ErrAccountCurrencyRequired
	}
	return nil
}
