package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/pp/types"
)

func TestParticipantResolveDIDOwner_IndexedOnCreate(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)

	id, err := k.CreateParticipant(ctx, types.Participant{Did: "did:example:alice", CorporationId: 7})
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)

	owner, found, err := k.ResolveDIDOwner(ctx, "did:example:alice")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(7), owner)

	_, found, err = k.ResolveDIDOwner(ctx, "did:example:absent")
	require.NoError(t, err)
	require.False(t, found)
}

// A second participant sharing the did under the same corp keeps a single owner.
func TestParticipantResolveDIDOwner_SharedDidSameCorp(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)

	_, err := k.CreateParticipant(ctx, types.Participant{Did: "did:example:x", CorporationId: 3})
	require.NoError(t, err)
	_, err = k.CreateParticipant(ctx, types.Participant{Did: "did:example:x", CorporationId: 3})
	require.NoError(t, err)

	owner, found, err := k.ResolveDIDOwner(ctx, "did:example:x")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(3), owner)
}
