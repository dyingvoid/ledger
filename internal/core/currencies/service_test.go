package currencies

import (
	"context"
	"errors"
	"testing"

	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const testCurrency = "USD"

type fakeRepository struct {
	createErr   error
	created     domain.CurrencyInfo
	createCalls int
}

func (f *fakeRepository) Create(_ context.Context, currency domain.CurrencyInfo) (domain.CurrencyInfo, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.CurrencyInfo{}, f.createErr
	}
	f.created = currency
	return currency, nil
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("creates a valid currency", func(t *testing.T) {
		t.Parallel()

		repository := &fakeRepository{}
		service := NewService(repository)

		created, err := service.Create(context.Background(), CreateParams{
			Code:       testCurrency,
			Name:       "US Dollar",
			Ledger:     1,
			AssetScale: 2,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.Code != "USD" {
			t.Fatalf("Code = %q, want USD", created.Code)
		}
		if repository.createCalls != 1 {
			t.Fatalf("Create() calls = %d, want 1", repository.createCalls)
		}
	})

	tests := []struct {
		name       string
		assetScale int32
		want       error
	}{
		{name: "rejects negative asset scale", assetScale: -1, want: domain_errors.ErrCurrencyAssetScaleInvalid},
		{name: "rejects asset scale above 18", assetScale: 19, want: domain_errors.ErrCurrencyAssetScaleInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{}
			service := NewService(repository)

			_, err := service.Create(context.Background(), CreateParams{
				Code:       testCurrency,
				Name:       "US Dollar",
				Ledger:     1,
				AssetScale: test.assetScale,
			})
			if !errors.Is(err, test.want) {
				t.Fatalf("Create() error = %v, want %v", err, test.want)
			}
			if repository.createCalls != 0 {
				t.Fatalf("Create() calls = %d, want 0", repository.createCalls)
			}
		})
	}

	t.Run("wraps repository error", func(t *testing.T) {
		t.Parallel()

		createErr := errors.New("postgres unavailable")
		repository := &fakeRepository{createErr: createErr}
		service := NewService(repository)

		_, err := service.Create(context.Background(), CreateParams{
			Code:       testCurrency,
			Name:       "US Dollar",
			Ledger:     1,
			AssetScale: 2,
		})
		if !errors.Is(err, createErr) {
			t.Fatalf("Create() error = %v, want %v", err, createErr)
		}
	})
}
