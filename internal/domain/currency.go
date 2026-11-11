package domain

import (
	"strings"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

type CurrencyInfo struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Ledger     uint32 `json:"ledger"`
	AssetScale int16  `json:"asset_scale"`
}

func NewCurrencyInfo(code string, name string, ledger uint32, assetScale int32) (CurrencyInfo, error) {
	if assetScale < 0 || assetScale > 18 {
		return CurrencyInfo{}, domain_errors.ErrCurrencyAssetScaleInvalid
	}

	info := CurrencyInfo{
		Code:       strings.TrimSpace(code),
		Name:       strings.TrimSpace(name),
		Ledger:     ledger,
		AssetScale: int16(assetScale),
	}
	if err := info.Validate(); err != nil {
		return CurrencyInfo{}, err
	}
	return info, nil
}

func (c CurrencyInfo) Validate() error {
	if strings.TrimSpace(c.Code) == "" {
		return domain_errors.ErrCurrencyCodeRequired
	}
	if strings.TrimSpace(c.Name) == "" {
		return domain_errors.ErrCurrencyNameRequired
	}
	if c.Ledger == 0 {
		return domain_errors.ErrCurrencyLedgerInvalid
	}
	if c.AssetScale < 0 || c.AssetScale > 18 {
		return domain_errors.ErrCurrencyAssetScaleInvalid
	}
	return nil
}
