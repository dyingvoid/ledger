package domain

import "errors"

var ErrAccountCurrencyRequired = errors.New("account currency is required")

type Currency string

type CurrencyInfo struct {
	Code       Currency
	Ledger     uint32
	AssetScale int16
}
