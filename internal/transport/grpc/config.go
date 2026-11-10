package grpc

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr string `envconfig:"ADDR" default:":50051"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("GRPC", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing grpc config: %w", err)
	}
	return cfg, nil
}
