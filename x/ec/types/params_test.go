package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ec/types"
)

func TestParams_EmptyRoundTrip(t *testing.T) {
	want := types.DefaultParams()
	bz, err := want.Marshal()
	require.NoError(t, err)
	require.Empty(t, bz)

	var got types.Params
	require.NoError(t, got.Unmarshal(bz))
	require.Equal(t, want, got)
	require.NoError(t, got.Validate())
}
