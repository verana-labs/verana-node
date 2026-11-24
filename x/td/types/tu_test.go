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
