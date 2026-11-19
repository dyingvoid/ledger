package repositories

import (
	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/domain"
)

func toTigerbeetleID(id domain.ID) tigerbeetle_go.Uint128 {
	return tigerbeetle_go.BigIntToUint128(id.BigInt())
}

func fromTigerbeetleID(id tigerbeetle_go.Uint128) domain.ID {
	var raw [16]byte
	id.BigInt().FillBytes(raw[:])
	return domain.IDFromBytes(raw)
}
