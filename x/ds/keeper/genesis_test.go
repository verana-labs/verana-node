package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestGenesis(t *testing.T) {
	f := initFixture(t)
	p := types.DefaultParams()
	p.CouncilBudget = math.LegacyNewDec(42)
	gs := types.GenesisState{Params: p}

	require.NoError(t, f.keeper.InitGenesis(f.ctx, gs))
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, gs.Params, got.Params)
}
