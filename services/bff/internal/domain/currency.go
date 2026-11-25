package domain

import (
	"errors"
	"strings"
)

var (
	ErrCurrencyCodeRequired      = errors.New("currency code is required")
	ErrCurrencyNameRequired      = errors.New("currency name is required")
	ErrCurrencyLedgerInvalid     = errors.New("currency ledger must be positive")
	ErrCurrencyAssetScaleInvalid = errors.New("currency asset scale must be between 0 and 18")
)

type Currency struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Ledger     uint32 `json:"ledger"`
	AssetScale int16  `json:"asset_scale"`
}

type CreateCurrencyParams struct {
	Code       string
	Name       string
	Ledger     uint32
	AssetScale int32
}

func NewCurrency(code string, name string, ledger uint32, assetScale int32) (Currency, error) {
	if ledger == 0 {
		return Currency{}, ErrCurrencyLedgerInvalid
	}
	if assetScale < 0 || assetScale > 18 {
		return Currency{}, ErrCurrencyAssetScaleInvalid
	}

	currency := Currency{
		Code:       strings.TrimSpace(code),
		Name:       strings.TrimSpace(name),
		Ledger:     ledger,
		AssetScale: int16(assetScale),
	}
	if currency.Code == "" {
		return Currency{}, ErrCurrencyCodeRequired
	}
	if currency.Name == "" {
		return Currency{}, ErrCurrencyNameRequired
	}
	return currency, nil
}
