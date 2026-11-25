package accounting

import (
	"context"
	"fmt"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	accountsv1 "github.com/dyingvoid/ledger/shared/pkg/proto/ledger/accounts/v1"
	currenciesv1 "github.com/dyingvoid/ledger/shared/pkg/proto/ledger/currencies/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	connection *grpc.ClientConn
	accounts   accountsv1.AccountServiceClient
	currencies currenciesv1.CurrencyServiceClient
}

func NewClient(cfg Config) (*Client, error) {
	connection, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connecting to accounting at %s: %w", cfg.Addr, err)
	}
	return &Client{
		connection: connection,
		accounts:   accountsv1.NewAccountServiceClient(connection),
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

func (c *Client) CreateAccountInfo(
	ctx context.Context,
	accountingID string,
	account domain.CreateAccountParams,
) error {
	_, err := c.accounts.CreateAccountInfo(ctx, &accountsv1.CreateAccountInfoRequest{
		Id:          accountingID,
		UserId:      account.UserID,
		Name:        account.Name,
		Description: account.Description,
		Type:        account.Type,
		Currency:    account.Currency,
	})
	if err != nil {
		return fmt.Errorf("creating account info in accounting: %w", err)
	}
	return nil
}

func (c *Client) OpenLedgerAccount(ctx context.Context, accountingID string) error {
	_, err := c.accounts.OpenLedgerAccount(ctx, &accountsv1.OpenLedgerAccountRequest{
		Id: accountingID,
	})
	if err != nil {
		return fmt.Errorf("opening ledger account in accounting: %w", err)
	}
	return nil
}
