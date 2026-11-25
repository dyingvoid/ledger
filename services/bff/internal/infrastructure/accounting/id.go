package accounting

import tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

type IDGenerator struct{}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

func (*IDGenerator) NewID() string {
	return tigerbeetle_go.ID().BigInt().String()
}
