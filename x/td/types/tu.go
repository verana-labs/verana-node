package types

import (
	"fmt"
	"math/big"

	collcodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TuValue stores a ledger value as the big-endian unsigned integer of the value scaled by 10^18.
var TuValue collcodec.ValueCodec[math.LegacyDec] = tuValueCodec{sdk.LegacyDecValue}

type tuValueCodec struct {
	collcodec.ValueCodec[math.LegacyDec]
}

func (tuValueCodec) Encode(v math.LegacyDec) ([]byte, error) {
	if v.IsNil() || v.IsNegative() {
		return nil, fmt.Errorf("invalid ledger value %v", v)
	}
	if v.IsZero() {
		return []byte{0}, nil
	}
	return v.BigInt().Bytes(), nil
}

func (tuValueCodec) Decode(b []byte) (math.LegacyDec, error) {
	return math.LegacyNewDecFromBigIntWithPrec(new(big.Int).SetBytes(b), math.LegacyPrecision), nil
}

func (tuValueCodec) ValueType() string {
	return "tu"
}
