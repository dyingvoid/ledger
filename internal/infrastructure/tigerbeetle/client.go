package tigerbeetle

import (
	"fmt"
	"math/big"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"
)

func NewClient(cfg Config) (tigerbeetle_go.Client, error) {
	clusterID, err := parseClusterID(cfg.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("parsing tigerbeetle cluster id: %w", err)
	}

	client, err := tigerbeetle_go.NewClient(clusterID, cfg.Addresses)
	if err != nil {
		return nil, fmt.Errorf("creating tigerbeetle client: %w", err)
	}

	return client, nil
}

func parseClusterID(value string) (tigerbeetle_go.Uint128, error) {
	id, ok := new(big.Int).SetString(value, 0)
	if !ok {
		return tigerbeetle_go.Uint128{}, fmt.Errorf("invalid value %q", value)
	}
	if id.BitLen() > 128 {
		return tigerbeetle_go.Uint128{}, fmt.Errorf("value %q exceeds 128 bits", value)
	}

	return tigerbeetle_go.BigIntToUint128(id), nil
}
