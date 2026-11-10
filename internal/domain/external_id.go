package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var ErrExternalIDInvalid = errors.New("external account ID is invalid")

const externalIDSize = 16

var maxExternalID = new(big.Int).Sub(
	new(big.Int).Lsh(big.NewInt(1), 128),
	big.NewInt(1),
)

type ExternalID [externalIDSize]byte

func ParseExternalID(value string) (ExternalID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ExternalID{}, fmt.Errorf("%w: empty", ErrExternalIDInvalid)
	}

	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		value = value[2:]
	}

	number, ok := new(big.Int).SetString(value, base)
	if !ok {
		return ExternalID{}, fmt.Errorf("%w: %q", ErrExternalIDInvalid, value)
	}
	if number.Sign() <= 0 {
		return ExternalID{}, fmt.Errorf("%w: must be positive", ErrExternalIDInvalid)
	}
	if number.BitLen() > 128 {
		return ExternalID{}, fmt.Errorf("%w: exceeds 128 bits", ErrExternalIDInvalid)
	}
	if number.Cmp(maxExternalID) == 0 {
		return ExternalID{}, fmt.Errorf("%w: must not be the maximum value", ErrExternalIDInvalid)
	}

	return externalIDFromBigInt(number), nil
}

func externalIDFromBigInt(number *big.Int) ExternalID {
	bigEndian := number.FillBytes(make([]byte, externalIDSize))

	var id ExternalID
	for i := range bigEndian {
		id[i] = bigEndian[externalIDSize-1-i]
	}
	return id
}

func (id ExternalID) Bytes() [16]byte {
	return [16]byte(id)
}

func (id ExternalID) BigInt() *big.Int {
	var bigEndian [externalIDSize]byte
	for i := range id {
		bigEndian[i] = id[externalIDSize-1-i]
	}
	return new(big.Int).SetBytes(bigEndian[:])
}

func (id ExternalID) String() string {
	return id.BigInt().String()
}

func (id ExternalID) IsZero() bool {
	return id == ExternalID{}
}
