package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/pp/keeper"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// Entries written before the indexes existed are picked up by the migration.
func TestMigrate1to2_FillsLookupIndexes(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)
	now := ctx.BlockTime()

	require.NoError(t, k.Participant.Set(ctx, 1, types.Participant{Id: 1, SchemaId: 3, CorporationId: 7, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:root"}))
	require.NoError(t, k.Participant.Set(ctx, 2, types.Participant{Id: 2, SchemaId: 3, CorporationId: 7, Role: types.ParticipantRole_ISSUER, Did: "did:example:slashed", Slashed: &now, SlashedDeposit: 100}))
	require.NoError(t, k.Participant.Set(ctx, 3, types.Participant{Id: 3, SchemaId: 3, CorporationId: 7, Role: types.ParticipantRole_ISSUER, Did: "did:example:repaid", Slashed: &now, SlashedDeposit: 100, RepaidDeposit: 100}))

	require.NoError(t, keeper.NewMigrator(k).Migrate1to2(ctx))

	has, err := k.RootParticipantBySchemaCorp.Has(ctx, collections.Join3(uint64(3), uint64(7), uint64(1)))
	require.NoError(t, err)
	require.True(t, has)
	has, err = k.UnrepaidSlashByCorp.Has(ctx, collections.Join3(uint64(7), uint64(3), uint64(2)))
	require.NoError(t, err)
	require.True(t, has)
	has, err = k.UnrepaidSlashByCorp.Has(ctx, collections.Join3(uint64(7), uint64(3), uint64(3)))
	require.NoError(t, err)
	require.False(t, has)
}
