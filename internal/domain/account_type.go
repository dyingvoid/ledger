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

func (t AccountType) Valid() bool {
	return t >= minAccountType && t <= maxAccountType
}

func (t AccountType) Validate() error {
	if !t.Valid() {
		return fmt.Errorf("%w: %d", domain_errors.ErrAccountTypeInvalid, t)
	}
	return nil
}

func (t AccountType) NormalBalance() (BalanceSide, error) {
	switch {
	case t >= 1000 && t <= 1999:
		return Debit, nil
	case t >= 2000 && t <= 4999:
		return Credit, nil
	case t >= 5000 && t <= 5999:
		return Debit, nil
	default:
		return 0, fmt.Errorf("%w: %d", domain_errors.ErrAccountTypeInvalid, t)
	}
}
