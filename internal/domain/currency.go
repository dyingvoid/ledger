package domain

import "errors"

var ErrAccountCurrencyRequired = errors.New("account currency is required")

type Currency string
