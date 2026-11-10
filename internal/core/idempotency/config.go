package idempotency

import (
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const DefaultTTL = 24 * time.Hour

var ErrTTLInvalid = errors.New("idempotency TTL must be positive")

type Config struct {
	TTL time.Duration `envconfig:"TTL" default:"24h"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("IDEMPOTENCY", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing idempotency config: %w", err)
	}
	if cfg.TTL <= 0 {
		return Config{}, ErrTTLInvalid
	}
	return cfg, nil
}
