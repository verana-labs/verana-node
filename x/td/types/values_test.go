package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/td/types"
)

func TestTuValue(t *testing.T) {
	for _, s := range []string{"0", "0.000000000000000001", "99.9", "1000", "123456789012345678901234567890.123456789012345678"} {
		d := math.LegacyMustNewDecFromStr(s)
		bz, err := types.TuValue.Encode(d)
		require.NoError(t, err)
		got, err := types.TuValue.Decode(bz)
		require.NoError(t, err)
		require.True(t, d.Equal(got), s)
	}

	bz, err := types.TuValue.Encode(math.LegacyNewDec(1000))
	require.NoError(t, err)
	require.Len(t, bz, 9)

	_, err = types.TuValue.Encode(math.LegacyNewDec(-1))
	require.Error(t, err)
}

func TestTsValue(t *testing.T) {
	for _, v := range []int64{0, 1_700_000_000, -62_135_596_800} {
		bz, err := types.TsValue.Encode(v)
		require.NoError(t, err)
		got, err := types.TsValue.Decode(bz)
		require.NoError(t, err)
		require.Equal(t, v, got)
	}

	bz, err := types.TsValue.Encode(1_700_000_000)
	require.NoError(t, err)
	require.Len(t, bz, 5)
}

func TestU64Value(t *testing.T) {
	for _, v := range []uint64{0, 41, 1<<64 - 1} {
		bz, err := types.U64Value.Encode(v)
		require.NoError(t, err)
		got, err := types.U64Value.Decode(bz)
		require.NoError(t, err)
		require.Equal(t, v, got)
	}

	bz, err := types.U64Value.Encode(41)
	require.NoError(t, err)
	require.Equal(t, []byte{41}, bz)
}

func TestU64Value_Invalid(t *testing.T) {
	for _, bz := range [][]byte{nil, {0x80}, {0x01, 0x00}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02}} {
		_, err := types.U64Value.Decode(bz)
		require.Error(t, err, "%x", bz)
		_, err = types.TsValue.Decode(bz)
		require.Error(t, err, "%x", bz)
	}
}
