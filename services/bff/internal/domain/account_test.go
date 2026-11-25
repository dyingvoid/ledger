package domain

import (
	"errors"
	"testing"
)

const accountName = "Main account"

func TestNewAccountOperationID(t *testing.T) {
	t.Parallel()

	first := NewAccountOperationID("create-main-account")
	second := NewAccountOperationID("create-main-account")
	different := NewAccountOperationID("create-second-account")

	if first != second {
		t.Fatalf("same idempotency key produced different operation ids: %q and %q", first, second)
	}
	if first == different {
		t.Fatalf("different idempotency keys produced the same operation id: %q", first)
	}
	if !IsAccountOperationID(first) {
		t.Fatalf("operation id %q is invalid", first)
	}
}

func TestNewCreateAccountParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      int64
		displayName string
		accountType int32
		currency    string
		wantErr     error
	}{
		{
			name:        "valid",
			userID:      42,
			displayName: " " + accountName + " ",
			accountType: 2100,
			currency:    " " + usdCode + " ",
		},
		{
			name:        "invalid user",
			userID:      0,
			displayName: accountName,
			accountType: 2100,
			currency:    usdCode,
			wantErr:     ErrAccountUserIDInvalid,
		},
		{
			name:        "missing name",
			userID:      42,
			displayName: " ",
			accountType: 2100,
			currency:    usdCode,
			wantErr:     ErrAccountNameRequired,
		},
		{
			name:        "invalid type",
			userID:      42,
			displayName: accountName,
			accountType: 999,
			currency:    usdCode,
			wantErr:     ErrAccountTypeInvalid,
		},
		{
			name:        "missing currency",
			userID:      42,
			displayName: accountName,
			accountType: 2100,
			currency:    " ",
			wantErr:     ErrAccountCurrencyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, err := NewCreateAccountParams(
				tt.userID,
				tt.displayName,
				"",
				tt.accountType,
				tt.currency,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewCreateAccountParams() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if params.Name != accountName {
					t.Fatalf("name = %q, want %s", params.Name, accountName)
				}
				if params.Currency != usdCode {
					t.Fatalf("currency = %q, want %s", params.Currency, usdCode)
				}
			}
		})
	}
}
