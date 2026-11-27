package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestGenesisState_Validate(t *testing.T) {
	require.NoError(t, types.DefaultGenesis().Validate())

	gs := types.DefaultGenesis()
	gs.Params.UnbondingPeriod = 0
	require.Error(t, gs.Validate())
}
