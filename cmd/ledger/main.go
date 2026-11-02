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

	"github.com/joho/godotenv"

	"github.com/dyingvoid/ledger/internal/infrastructure/postgres"
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
