package domain

import (
	"fmt"
	"math/big"
	"strings"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

const idSize = 16

var maxID = new(big.Int).Sub(
	new(big.Int).Lsh(big.NewInt(1), 128),
	big.NewInt(1),
)

type ID [idSize]byte

func ParseID(value string) (ID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ID{}, fmt.Errorf("%w: empty", domain_errors.ErrIDInvalid)
	}

	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		value = value[2:]
	}

	number, ok := new(big.Int).SetString(value, base)
	if !ok {
		return ID{}, fmt.Errorf("%w: %q", domain_errors.ErrIDInvalid, value)
	}
	if number.Sign() <= 0 {
		return ID{}, fmt.Errorf("%w: must be positive", domain_errors.ErrIDInvalid)
	}
	if number.BitLen() > 128 {
		return ID{}, fmt.Errorf("%w: exceeds 128 bits", domain_errors.ErrIDInvalid)
	}
	if number.Cmp(maxID) == 0 {
		return ID{}, fmt.Errorf("%w: must not be the maximum value", domain_errors.ErrIDInvalid)
	}

	var raw [idSize]byte
	number.FillBytes(raw[:])
	return IDFromBytes(raw), nil
}

func IDFromBytes(value [idSize]byte) ID {
	return ID(value)
}

func (id ID) Bytes() [idSize]byte {
	return [idSize]byte(id)
}

func (id ID) BigInt() *big.Int {
	return new(big.Int).SetBytes(id[:])
}

func (id ID) String() string {
	return id.BigInt().String()
}

func (id ID) IsZero() bool {
	return id == ID{}
}
