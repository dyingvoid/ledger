package currencies

import (
	"context"
	"testing"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
)

type fakeGateway struct {
	currency domain.Currency
}

func (f *fakeGateway) CreateCurrency(_ context.Context, currency domain.Currency) (domain.Currency, error) {
	f.currency = currency
	return currency, nil
}

func TestServiceCreateCurrency(t *testing.T) {
	t.Parallel()

	gateway := &fakeGateway{}
	service := NewService(gateway)

	created, err := service.CreateCurrency(context.Background(), domain.CreateCurrencyParams{
		Code:       " USD ",
		Name:       "US Dollar",
		Ledger:     1,
		AssetScale: 2,
	})
	if err != nil {
		t.Fatalf("CreateCurrency() error = %v", err)
	}
	if created.Code != "USD" {
		t.Fatalf("CreateCurrency() code = %q, want USD", created.Code)
	}
	if gateway.currency.Name != "US Dollar" {
		t.Fatalf("gateway name = %q, want US Dollar", gateway.currency.Name)
	}
}
