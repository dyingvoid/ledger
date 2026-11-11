package domain_errors

import "errors"

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
