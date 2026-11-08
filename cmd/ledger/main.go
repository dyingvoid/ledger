package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres/repositories"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("loading .env: %w", err)
	}

	location, err := loadLocation()
	if err != nil {
		return err
	}
	time.Local = location

	cfg, err := postgres.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading postgres config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	services, err := newServices(pool)
	if err != nil {
		return fmt.Errorf("building services: %w", err)
	}

	return serve(ctx, services)
}

type services struct {
	accounts      *accounts.Service
	idempotency   *idempotency.Service
	createAccount *idempotency.Runner[domain.Account]
}

func newServices(pool *pgxpool.Pool) (*services, error) {
	idempotencyCfg, err := idempotency.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading idempotency config: %w", err)
	}

	accountRepository := repositories.NewAccounts(pool)
	idempotencyStore := repositories.NewIdempotency(pool)
	unitOfWork := postgres.NewUnitOfWork(pool)
	idempotencyService := idempotency.NewService(idempotencyStore, unitOfWork, idempotencyCfg)

	return &services{
		accounts:      accounts.NewService(accountRepository),
		idempotency:   idempotencyService,
		createAccount: idempotency.NewRunner[domain.Account](idempotencyService),
	}, nil
}

func serve(ctx context.Context, services *services) error {
	if services == nil ||
		services.accounts == nil ||
		services.idempotency == nil ||
		services.createAccount == nil {
		return errors.New("services are not initialized")
	}

	<-ctx.Done()

	return nil
}

func loadLocation() (*time.Location, error) {
	name := os.Getenv("TIME_ZONE")
	if name == "" {
		return time.UTC, nil
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid TIME_ZONE %q: %w", name, err)
	}

	return location, nil
}
