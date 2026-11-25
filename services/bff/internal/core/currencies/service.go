package currencies

import (
	"context"
	"fmt"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
)

type Gateway interface {
	CreateCurrency(ctx context.Context, currency domain.Currency) (domain.Currency, error)
}

type Service struct {
	gateway Gateway
}

func NewService(gateway Gateway) *Service {
	return &Service{gateway: gateway}
}

func (s *Service) CreateCurrency(
	ctx context.Context,
	params domain.CreateCurrencyParams,
) (domain.Currency, error) {
	currency, err := domain.NewCurrency(params.Code, params.Name, params.Ledger, params.AssetScale)
	if err != nil {
		return domain.Currency{}, fmt.Errorf("validating currency: %w", err)
	}

	created, err := s.gateway.CreateCurrency(ctx, currency)
	if err != nil {
		return domain.Currency{}, fmt.Errorf("creating currency: %w", err)
	}
	return created, nil
}
