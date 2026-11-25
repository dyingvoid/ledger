package domain_errors

import "errors"

var (
	ErrDepositIDRequired                   = errors.New("deposit ID is required")
	ErrDepositAmountInvalid                = errors.New("deposit amount must be positive")
	ErrDepositCurrencyRequired             = errors.New("deposit currency is required")
	ErrDepositClientAccountRequired        = errors.New("deposit client account is required")
	ErrDepositBankAccountRequired          = errors.New("deposit bank account is required")
	ErrDepositAccountsIdentical            = errors.New("deposit client and bank accounts must differ")
	ErrDepositAccountDeleted               = errors.New("deposit account is deleted")
	ErrDepositClientAccountNotFound        = errors.New("deposit client account not found")
	ErrDepositBankAccountNotFound          = errors.New("deposit bank account not found")
	ErrDepositClientAccountTypeUnsupported = errors.New("account type does not support client deposits")
	ErrDepositBankAccountTypeUnsupported   = errors.New("account type cannot fund deposits")
	ErrDepositClientCurrencyMismatch       = errors.New("client account currency does not match deposit currency")
	ErrDepositBankCurrencyMismatch         = errors.New("bank account currency does not match deposit currency")
	ErrDepositCurrencyMismatch             = errors.New("deposit accounts have different ledgers")
	ErrDepositConflict                     = errors.New("deposit conflicts with an existing transfer")
)
