package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
)

func TestCoResolveDIDOwner(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	require.NoError(t, k.CorporationByDID.Set(ctx, "did:example:co", 42))

	id, found, err := k.ResolveDIDOwner(ctx, "did:example:co")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(42), id)

	_, found, err = k.ResolveDIDOwner(ctx, "did:example:absent")
	require.NoError(t, err)
	require.False(t, found)
}
