package domain

import (
	"errors"
	"testing"
)

const (
	usdCode = "USD"
	usdName = "US Dollar"
)

func TestNewCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        string
		displayName string
		ledger      uint32
		assetScale  int32
		wantErr     error
	}{
		{name: "valid", code: usdCode, displayName: usdName, ledger: 1, assetScale: 2},
		{name: "missing code", code: " ", displayName: usdName, ledger: 1, assetScale: 2, wantErr: ErrCurrencyCodeRequired},
		{name: "missing name", code: usdCode, displayName: " ", ledger: 1, assetScale: 2, wantErr: ErrCurrencyNameRequired},
		{name: "invalid ledger", code: usdCode, displayName: usdName, ledger: 0, assetScale: 2, wantErr: ErrCurrencyLedgerInvalid},
		{name: "invalid scale", code: usdCode, displayName: usdName, ledger: 1, assetScale: 19, wantErr: ErrCurrencyAssetScaleInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			currency, err := NewCurrency(tt.code, tt.displayName, tt.ledger, tt.assetScale)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewCurrency() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && currency.Code != usdCode {
				t.Fatalf("NewCurrency() code = %q, want %s", currency.Code, usdCode)
			}
		})
	}
}
