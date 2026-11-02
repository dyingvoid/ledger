package postgres

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DSN             string        `envconfig:"DSN" required:"true"`
	MaxConns        int32         `envconfig:"MAX_CONNS" default:"10"`
	MinConns        int32         `envconfig:"MIN_CONNS" default:"0"`
	MaxConnLifetime time.Duration `envconfig:"MAX_CONN_LIFETIME" default:"1h"`
	MaxConnIdleTime time.Duration `envconfig:"MAX_CONN_IDLE_TIME" default:"30m"`
	ConnectTimeout  time.Duration `envconfig:"CONNECT_TIMEOUT" default:"5s"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("POSTGRES", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing postgres config: %w", err)
	}
	return cfg, nil
}
