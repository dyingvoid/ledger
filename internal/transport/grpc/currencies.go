package grpc

import (
	"context"

	"github.com/dyingvoid/ledger/internal/core/currencies"
	currenciesv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/currencies/v1"
)

type currencyService struct {
	currenciesv1.UnimplementedCurrencyServiceServer
	currencies *currencies.Service
}

func newCurrencyService(currenciesService *currencies.Service) *currencyService {
	return &currencyService{currencies: currenciesService}
}

func (s *currencyService) CreateCurrency(
	ctx context.Context,
	req *currenciesv1.CreateCurrencyRequest,
) (*currenciesv1.CreateCurrencyResponse, error) {
	currency, err := s.currencies.Create(ctx, currencies.CreateParams{
		Code:       req.GetCode(),
		Name:       req.GetName(),
		Ledger:     req.GetLedger(),
		AssetScale: req.GetAssetScale(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &currenciesv1.CreateCurrencyResponse{
		Currency: &currenciesv1.CurrencyInfo{
			Code:       currency.Code,
			Name:       currency.Name,
			Ledger:     currency.Ledger,
			AssetScale: int32(currency.AssetScale),
		},
	}, nil
}
