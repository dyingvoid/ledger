package domain

import (
	"fmt"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const (
	AccountTypeBankCash      AccountType = 1000
	AccountTypeClientDeposit AccountType = 2100

	minAccountType AccountType = 1000
	maxAccountType AccountType = 5999
)

type AccountType int32

type AccountCategory uint8

const (
	AccountCategoryAsset AccountCategory = iota + 1
	AccountCategoryLiability
	AccountCategoryEquity
	AccountCategoryRevenue
	AccountCategoryExpense
)

func (t AccountType) Validate() error {
	if t < minAccountType || t > maxAccountType {
		return fmt.Errorf("%w: %d", domain_errors.ErrAccountTypeInvalid, t)
	}
	return nil
}

func (t AccountType) Category() (AccountCategory, error) {
	switch {
	case t >= 1000 && t <= 1999:
		return AccountCategoryAsset, nil
	case t >= 2000 && t <= 2999:
		return AccountCategoryLiability, nil
	case t >= 3000 && t <= 3999:
		return AccountCategoryEquity, nil
	case t >= 4000 && t <= 4999:
		return AccountCategoryRevenue, nil
	case t >= 5000 && t <= 5999:
		return AccountCategoryExpense, nil
	default:
		return 0, fmt.Errorf("%w: %d", domain_errors.ErrAccountTypeInvalid, t)
	}
}

func (t AccountType) SupportsDeposits() bool {
	return t == AccountTypeClientDeposit
}

func (t AccountType) CanFundDeposits() bool {
	return t == AccountTypeBankCash
}

func (t AccountType) NormalBalance() (BalanceSide, error) {
	category, err := t.Category()
	if err != nil {
		return 0, err
	}

	switch category {
	case AccountCategoryAsset, AccountCategoryExpense:
		return Debit, nil
	case AccountCategoryLiability, AccountCategoryEquity, AccountCategoryRevenue:
		return Credit, nil
	default:
		return 0, fmt.Errorf("%w: %d", domain_errors.ErrAccountTypeInvalid, t)
	}
}
