package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	accountsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/accounts/v1"
	currenciesv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/currencies/v1"
)

const (
	bankUserID      int64 = 1
	firstClientUser int64 = 1000
)

type seedCurrency struct {
	code       string
	name       string
	ledger     uint32
	assetScale int16
}

var seedCurrencies = []seedCurrency{
	{code: "USD", name: "US Dollar", ledger: 1, assetScale: 2},
	{code: "CNY", name: "Chinese Yuan", ledger: 2, assetScale: 2},
	{code: "RUB", name: "Russian Ruble", ledger: 3, assetScale: 2},
}

type seeder struct {
	accounts   accountsv1.AccountServiceClient
	currencies currenciesv1.CurrencyServiceClient
}

func main() {
	addr := flag.String("addr", "localhost:50051", "ledger grpc address")
	users := flag.Int("users", 5, "number of client users to seed")
	accountsPerUser := flag.Int("accounts-per-user", 1, "client accounts per user per currency")
	flag.Parse()

	if err := run(*addr, *users, *accountsPerUser); err != nil {
		log.Fatal(err)
	}
}

func run(addr string, users int, accountsPerUser int) error {
	connection, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}
	defer func() {
		_ = connection.Close()
	}()

	s := &seeder{
		accounts:   accountsv1.NewAccountServiceClient(connection),
		currencies: currenciesv1.NewCurrencyServiceClient(connection),
	}

	ctx := context.Background()
	for _, currency := range seedCurrencies {
		if err := s.createCurrency(ctx, currency); err != nil {
			return err
		}
		if err := s.createBankCash(ctx, currency); err != nil {
			return err
		}
		for user := range users {
			for account := range accountsPerUser {
				if err := s.createClientAccount(ctx, currency, int64(user), account); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *seeder) createCurrency(ctx context.Context, currency seedCurrency) error {
	response, err := s.currencies.CreateCurrency(ctx, &currenciesv1.CreateCurrencyRequest{
		Code:       currency.code,
		Name:       currency.name,
		Ledger:     currency.ledger,
		AssetScale: int32(currency.assetScale),
	})
	if err != nil {
		return fmt.Errorf("creating currency %s: %w", currency.code, err)
	}

	log.Printf("currency %s ledger=%d", response.GetCurrency().GetCode(), response.GetCurrency().GetLedger())
	return nil
}

func (s *seeder) createBankCash(ctx context.Context, currency seedCurrency) error {
	id := deterministicID("ledger.bank-cash.v1", strconv.FormatUint(uint64(currency.ledger), 10))
	if err := s.createAccount(
		ctx,
		id,
		bankUserID,
		"Bank Cash "+currency.code,
		"Bank cash account for "+currency.code,
		int32(1000),
		currency.code,
	); err != nil {
		return err
	}

	return nil
}

func (s *seeder) createClientAccount(
	ctx context.Context,
	currency seedCurrency,
	user int64,
	account int,
) error {
	id := deterministicID(
		"ledger.seed.client.v1",
		strconv.FormatInt(firstClientUser+user, 10)+":"+currency.code+":"+strconv.Itoa(account),
	)
	name := fmt.Sprintf("User %d %s %d", firstClientUser+user, currency.code, account+1)

	if err := s.createAccount(
		ctx,
		id,
		firstClientUser+user,
		name,
		"Seeded client deposit account",
		int32(2100),
		currency.code,
	); err != nil {
		return err
	}

	return nil
}

func (s *seeder) createAccount(
	ctx context.Context,
	id string,
	userID int64,
	name string,
	description string,
	accountType int32,
	currency string,
) error {
	created, err := s.accounts.CreateAccountInfo(ctx, &accountsv1.CreateAccountInfoRequest{
		Id:          id,
		UserId:      userID,
		Name:        name,
		Description: description,
		Type:        accountType,
		Currency:    currency,
	})
	if err != nil {
		return fmt.Errorf("creating account %s: %w", id, err)
	}

	if _, err := s.accounts.OpenLedgerAccount(ctx, &accountsv1.OpenLedgerAccountRequest{Id: id}); err != nil {
		return fmt.Errorf("opening ledger account %s: %w", id, err)
	}

	log.Printf(
		"account %s user=%d type=%d currency=%s",
		created.GetAccount().GetId(),
		created.GetAccount().GetUserId(),
		accountType,
		created.GetAccount().GetCurrency(),
	)
	return nil
}

func deterministicID(namespace string, value string) string {
	digest := sha256.Sum256([]byte(namespace + ":" + value))
	return "0x" + hex.EncodeToString(digest[:16])
}
