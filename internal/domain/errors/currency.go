package domain_errors

import "errors"

var (
	ErrAccountCurrencyRequired   = errors.New("account currency is required")
	ErrCurrencyCodeRequired      = errors.New("currency code is required")
	ErrCurrencyNameRequired      = errors.New("currency name is required")
	ErrCurrencyLedgerInvalid     = errors.New("currency ledger must be positive")
	ErrCurrencyAssetScaleInvalid = errors.New("currency asset scale must be between 0 and 18")
)
