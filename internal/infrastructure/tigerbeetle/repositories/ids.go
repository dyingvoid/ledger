package repositories

import (
	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/dyingvoid/ledger/internal/domain"
)

func toTigerbeetleID(id domain.ID) tigerbeetle_go.Uint128 {
	return tigerbeetle_go.BigIntToUint128(id.BigInt())
}
