package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/vs/types"
)

func TestGenesis(t *testing.T) {
	f := initFixture(t)
	gs := types.GenesisState{Params: types.NewParams(1, 2, time.Hour, time.Minute)}

	require.NoError(t, f.keeper.InitGenesis(f.ctx, gs))
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, gs.Params, got.Params)
}
