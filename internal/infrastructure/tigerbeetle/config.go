package tigerbeetle

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ClusterID string   `envconfig:"CLUSTER_ID" default:"0"`
	Addresses []string `envconfig:"ADDRESSES" required:"true"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("TIGERBEETLE", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing tigerbeetle config: %w", err)
	}
	return cfg, nil
}
