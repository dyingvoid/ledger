package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/dyingvoid/ledger/services/bff/internal/core/currencies"
	"github.com/dyingvoid/ledger/services/bff/internal/infrastructure/accounting"
	transporthttp "github.com/dyingvoid/ledger/services/bff/internal/transport/http"
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

	location, err := loadLocation()
	if err != nil {
		return err
	}
	time.Local = location

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	httpCfg, err := transporthttp.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading http config: %w", err)
	}
	accountingCfg, err := accounting.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading accounting config: %w", err)
	}

	client, err := accounting.NewClient(accountingCfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Error("closing accounting client", "error", err)
		}
	}()

	service := currencies.NewService(client)
	handler := transporthttp.NewHandler(service, log)
	server := transporthttp.NewServer(httpCfg, handler.Routes())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return serve(ctx, server, log)
}

func serve(ctx context.Context, server *transporthttp.Server, log *slog.Logger) error {
	log.Info("starting bff server")

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down bff server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down bff server: %w", err)
	}
	err := <-errCh
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
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
