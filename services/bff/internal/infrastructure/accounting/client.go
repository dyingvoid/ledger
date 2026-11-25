package accounting

import (
	"context"
	"fmt"

	currenciesv1 "github.com/dyingvoid/ledger/services/accounting/pkg/proto/ledger/currencies/v1"
	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	connection *grpc.ClientConn
	currencies currenciesv1.CurrencyServiceClient
}

func NewClient(cfg Config) (*Client, error) {
	connection, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connecting to accounting at %s: %w", cfg.Addr, err)
	}
	return &Client{
		connection: connection,
		currencies: currenciesv1.NewCurrencyServiceClient(connection),
	}, nil
}

func (c *Client) Close() error {
	return c.connection.Close()
}

func (c *Client) CreateCurrency(ctx context.Context, currency domain.Currency) (domain.Currency, error) {
	response, err := c.currencies.CreateCurrency(ctx, &currenciesv1.CreateCurrencyRequest{
		Code:       currency.Code,
		Name:       currency.Name,
		Ledger:     currency.Ledger,
		AssetScale: int32(currency.AssetScale),
	})
	if err != nil {
		return domain.Currency{}, fmt.Errorf("creating currency in accounting: %w", err)
	}

	result := response.GetCurrency()
	currency, err = domain.NewCurrency(
		result.GetCode(),
		result.GetName(),
		result.GetLedger(),
		result.GetAssetScale(),
	)
	if err != nil {
		return domain.Currency{}, fmt.Errorf("validating accounting currency: %w", err)
	}
	return currency, nil
}
