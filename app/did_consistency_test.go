package app

import (
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	ectypes "github.com/verana-labs/verana-node/x/ec/types"
)

func TestAssertGenesisDIDConsistency(t *testing.T) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = uint(0)

	a, err := New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := a.NewContextLegacy(true, cmtproto.Header{})

	// Clean state passes.
	require.NoError(t, a.assertGenesisDIDConsistency(ctx))

	// Corporation 1 and an ecosystem of corporation 2 both claim the same did,
	// with no participant involved: this cross-type conflict must be rejected.
	require.NoError(t, a.CoKeeper.Corporation.Set(ctx, 1, cotypes.Corporation{Id: 1, Did: "did:dup"}))
	require.NoError(t, a.EcosystemKeeper.Ecosystem.Set(ctx, 5, ectypes.Ecosystem{Id: 5, Did: "did:dup", CorporationId: 2}))

	err = a.assertGenesisDIDConsistency(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "did:dup")
}
