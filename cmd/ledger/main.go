package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/currencies"
	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres/repositories"
	"github.com/dyingvoid/ledger/internal/infrastructure/tigerbeetle"
	tbrepositories "github.com/dyingvoid/ledger/internal/infrastructure/tigerbeetle/repositories"
	"github.com/dyingvoid/ledger/internal/logger"
	transportgrpc "github.com/dyingvoid/ledger/internal/transport/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("loading .env: %w", err)
	}

	logCfg, err := logger.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading logger config: %w", err)
	}

	log, err := logger.New(logCfg)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	slog.SetDefault(log)

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

	tbCfg, err := tigerbeetle.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading tigerbeetle config: %w", err)
	}

	tbClient, err := tigerbeetle.NewClient(tbCfg)
	if err != nil {
		return fmt.Errorf("connecting to tigerbeetle: %w", err)
	}
	defer tbClient.Close()

	services := newServices(pool, tbClient)

	grpcCfg, err := transportgrpc.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading grpc config: %w", err)
	}

	server, err := transportgrpc.NewServer(
		ctx,
		grpcCfg,
		log,
		services.accounts,
		services.currencies,
		services.deposits,
	)
	if err != nil {
		return fmt.Errorf("creating grpc server: %w", err)
	}

	return serve(ctx, server, log)
}

type services struct {
	accounts   *accounts.Service
	currencies *currencies.Service
	deposits   *deposits.Service
}

func newServices(pool *pgxpool.Pool, tbClient tigerbeetle_go.Client) *services {
	currencyInfos := repositories.NewCurrencyInfos(pool)
	accountInfos := repositories.NewAccounts(pool)

	accountLedger := tbrepositories.NewAccounts(tbClient, currencyInfos)
	depositLedger := tbrepositories.NewDeposits(tbClient)

	return &services{
		accounts:   accounts.NewService(accountInfos, accountLedger),
		currencies: currencies.NewService(currencyInfos),
		deposits:   deposits.NewService(accountInfos, currencyInfos, depositLedger),
	}
}

func serve(ctx context.Context, server *transportgrpc.Server, log *slog.Logger) error {
	log.Info("starting grpc server", "addr", server.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down grpc server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down grpc server: %w", err)
	}

	log.Info("grpc server stopped")

	return <-errCh
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
