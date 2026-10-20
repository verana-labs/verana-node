package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/keeper"
	"github.com/verana-labs/verana-node/x/de/types"
)

// Authorizations written before the index existed are picked up by the migration.
func TestMigrate1to2_FillsOperatorIndex(t *testing.T) {
	f := initFixture(t)
	k := f.keeper
	vsOp := acc("vsop_migrate________")
	require.NoError(t, k.VSOperatorAuthorizations.Set(f.ctx, 1, types.VSOperatorAuthorization{
		Id: 1, CorporationId: 4, VsOperator: vsOp,
		Records: []types.ParticipantAuthorizationRecord{{ParticipantId: 9, MsgTypes: []string{mtValidated}}},
	}))

	require.NoError(t, keeper.NewMigrator(k).Migrate1to2(sdk.UnwrapSDKContext(f.ctx)))

	corp, err := k.VSOACorpByOperator.Get(f.ctx, vsOp)
	require.NoError(t, err)
	require.Equal(t, uint64(4), corp)
}
