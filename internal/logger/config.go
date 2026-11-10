package logger

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Level  string `envconfig:"LEVEL" default:"info"`
	Format string `envconfig:"FORMAT" default:"json"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("LOG", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing logger config: %w", err)
	}
	return cfg, nil
}
