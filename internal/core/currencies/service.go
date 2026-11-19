package currencies

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

// TODO: to errors.go
var ErrCurrencyConflict = errors.New("currency already exists with different data")

type Repository interface {
	Create(ctx context.Context, currency domain.CurrencyInfo) (domain.CurrencyInfo, error)
}

type Service struct {
	currencies Repository
}

func NewService(currencies Repository) *Service {
	return &Service{currencies: currencies}
}

type CreateParams struct {
	Code       string
	Name       string
	Ledger     uint32
	AssetScale int32
}

func (s *Service) Create(ctx context.Context, params CreateParams) (domain.CurrencyInfo, error) {
	currency, err := domain.NewCurrencyInfo(
		params.Code,
		params.Name,
		params.Ledger,
		params.AssetScale,
	)
	if err != nil {
		return domain.CurrencyInfo{}, fmt.Errorf("validating currency: %w", err)
	}

	created, err := s.currencies.Create(ctx, currency)
	if err != nil {
		return domain.CurrencyInfo{}, fmt.Errorf("creating currency: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"currency created",
		"code", created.Code,
		"ledger", created.Ledger,
	)

	return created, nil
}
