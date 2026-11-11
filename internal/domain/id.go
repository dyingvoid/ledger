package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var ErrIDInvalid = errors.New("ID is invalid")

const idSize = 16

var maxID = new(big.Int).Sub(
	new(big.Int).Lsh(big.NewInt(1), 128),
	big.NewInt(1),
)

// ID is a 128-bit identifier shared by Postgres and TigerBeetle. Its raw form
// is little-endian because that is the representation TigerBeetle uses.
type ID [idSize]byte

func ParseID(value string) (ID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ID{}, fmt.Errorf("%w: empty", ErrIDInvalid)
	}

	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		value = value[2:]
	}

	number, ok := new(big.Int).SetString(value, base)
	if !ok {
		return ID{}, fmt.Errorf("%w: %q", ErrIDInvalid, value)
	}
	if number.Sign() <= 0 {
		return ID{}, fmt.Errorf("%w: must be positive", ErrIDInvalid)
	}
	if number.BitLen() > 128 {
		return ID{}, fmt.Errorf("%w: exceeds 128 bits", ErrIDInvalid)
	}
	if number.Cmp(maxID) == 0 {
		return ID{}, fmt.Errorf("%w: must not be the maximum value", ErrIDInvalid)
	}

	return idFromBigEndian(number.FillBytes(make([]byte, idSize))), nil
}

func IDFromBytes(value [idSize]byte) ID {
	return ID(value)
}

// Bytes returns the little-endian representation used by TigerBeetle.
func (id ID) Bytes() [idSize]byte {
	return [idSize]byte(id)
}

// BigEndianBytes returns the canonical big-endian representation used by
// Postgres.
func (id ID) BigEndianBytes() [idSize]byte {
	var bigEndian [idSize]byte
	for i := range id {
		bigEndian[i] = id[idSize-1-i]
	}
	return bigEndian
}

// IDFromBigEndianBytes converts a big-endian 128-bit value, such as the one
// stored by Postgres, into an ID.
func IDFromBigEndianBytes(value [idSize]byte) ID {
	return idFromBigEndian(value[:])
}

func idFromBigEndian(bigEndian []byte) ID {
	var id ID
	for i := range bigEndian {
		id[i] = bigEndian[idSize-1-i]
	}
	return id
}

func (id ID) BigInt() *big.Int {
	return new(big.Int).SetBytes(reversed(id.Bytes()))
}

func (id ID) String() string {
	return id.BigInt().String()
}

func (id ID) IsZero() bool {
	return id == ID{}
}

func reversed(value [idSize]byte) []byte {
	for i, j := 0, len(value)-1; i < j; i, j = i+1, j-1 {
		value[i], value[j] = value[j], value[i]
	}
	return value[:]
}
