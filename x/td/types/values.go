package types

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TuValue stores a ledger value as the big-endian unsigned integer of the value scaled by 10^18.
var TuValue collcodec.ValueCodec[math.LegacyDec] = tuValueCodec{sdk.LegacyDecValue}

// TsValue stores Unix seconds as an int64 varint.
var TsValue collcodec.ValueCodec[int64] = tsValueCodec{collections.Int64Value}

// U64Value stores a uint64 as a varint.
var U64Value collcodec.ValueCodec[uint64] = u64ValueCodec{collections.Uint64Value}

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

type tsValueCodec struct {
	collcodec.ValueCodec[int64]
}

func (tsValueCodec) Encode(v int64) ([]byte, error) {
	return binary.AppendUvarint(nil, uint64(v)), nil
}

func (tsValueCodec) Decode(b []byte) (int64, error) {
	v, err := decodeUvarint(b)
	return int64(v), err
}

func (tsValueCodec) ValueType() string {
	return "ts"
}

type u64ValueCodec struct {
	collcodec.ValueCodec[uint64]
}

func (u64ValueCodec) Encode(v uint64) ([]byte, error) {
	return binary.AppendUvarint(nil, v), nil
}

func (u64ValueCodec) Decode(b []byte) (uint64, error) {
	return decodeUvarint(b)
}

func (u64ValueCodec) ValueType() string {
	return "u64"
}

func decodeUvarint(b []byte) (uint64, error) {
	v, n := binary.Uvarint(b)
	if n <= 0 || n != len(b) {
		return 0, fmt.Errorf("invalid varint %x", b)
	}
	return v, nil
}
