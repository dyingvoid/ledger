package domain

import (
	"errors"
	"fmt"
	"strings"
)

type AccountType uint8

const (
	AccountTypeUnknown AccountType = iota
	AccountTypeAsset
	AccountTypeLiability
	AccountTypeEquity
	AccountTypeRevenue
	AccountTypeExpense
)

type BalanceSide uint8

const (
	Debit BalanceSide = iota + 1
	Credit
)

var (
	ErrAccountTypeUnknown  = errors.New("account ID has no known account type")
	ErrAccountNameRequired = errors.New("account name is required")
)

type Account struct {
	ID          int64
	Name        string
	Description string
}

func NewAccount(id int64, name, description string) (Account, error) {
	account := Account{
		ID:          id,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	}
	if err := account.Validate(); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (a Account) Type() AccountType {
	return accountType(a.ID)
}

func (a Account) NormalBalance() (BalanceSide, error) {
	switch a.Type() {
	case AccountTypeAsset, AccountTypeExpense:
		return Debit, nil
	case AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue:
		return Credit, nil
	default:
		return 0, fmt.Errorf("%w: %d", ErrAccountTypeUnknown, a.ID)
	}
}

func (a Account) Validate() error {
	if err := validateAccountID(a.ID); err != nil {
		return err
	}
	if strings.TrimSpace(a.Name) == "" {
		return ErrAccountNameRequired
	}
	return nil
}

func accountType(id int64) AccountType {
	switch {
	case id >= 1000 && id <= 1999:
		return AccountTypeAsset
	case id >= 2000 && id <= 2999:
		return AccountTypeLiability
	case id >= 3000 && id <= 3999:
		return AccountTypeEquity
	case id >= 4000 && id <= 4999:
		return AccountTypeRevenue
	case id >= 5000 && id <= 5999:
		return AccountTypeExpense
	default:
		return AccountTypeUnknown
	}
}

func validateAccountID(id int64) error {
	if accountType(id) == AccountTypeUnknown {
		return fmt.Errorf("%w: %d", ErrAccountTypeUnknown, id)
	}
	return nil
}
