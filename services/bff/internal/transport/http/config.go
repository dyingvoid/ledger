package http

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr              string        `envconfig:"ADDR" default:":8080"`
	ReadHeaderTimeout time.Duration `envconfig:"READ_HEADER_TIMEOUT" default:"5s"`
	ReadTimeout       time.Duration `envconfig:"READ_TIMEOUT" default:"10s"`
	WriteTimeout      time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout       time.Duration `envconfig:"IDLE_TIMEOUT" default:"60s"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("HTTP", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing http config: %w", err)
	}
	return cfg, nil
}
