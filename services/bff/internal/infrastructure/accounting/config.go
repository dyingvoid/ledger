package accounting

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr string `envconfig:"ADDR" default:"localhost:50051"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("ACCOUNTING", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing accounting config: %w", err)
	}
	return cfg, nil
}
