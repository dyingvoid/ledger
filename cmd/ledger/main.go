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

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
	"github.com/dyingvoid/ledger/internal/infrastructure/postgres/repositories"
	"github.com/dyingvoid/ledger/internal/infrastructure/tigerbeetle"
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

	services, err := newServices(pool)
	if err != nil {
		return fmt.Errorf("building services: %w", err)
	}

	grpcCfg, err := transportgrpc.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading grpc config: %w", err)
	}

	server, err := transportgrpc.NewServer(
		ctx,
		grpcCfg,
		log,
		services.accounts,
		services.deposits,
		services.idempotencyService,
	)
	if err != nil {
		return fmt.Errorf("creating grpc server: %w", err)
	}

	return serve(ctx, server, log)
}

type services struct {
	accounts           *accounts.Service
	deposits           *deposits.Service
	idempotencyService *idempotency.Service
}

func newServices(pool *pgxpool.Pool) (*services, error) {
	idempotencyCfg, err := idempotency.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading idempotency config: %w", err)
	}

	accountRepository := repositories.NewAccounts(pool)
	currencyRepository := repositories.NewCurrencies(pool)
	depositRepository := repositories.NewDeposits(pool)
	idempotencyStore := repositories.NewIdempotency(pool)
	unitOfWork := postgres.NewUnitOfWork(pool)
	idempotencyService := idempotency.NewService(idempotencyStore, unitOfWork, idempotencyCfg)

	return &services{
		accounts:           accounts.NewService(accountRepository, currencyRepository),
		deposits:           deposits.NewService(depositRepository),
		idempotencyService: idempotencyService,
	}, nil
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
