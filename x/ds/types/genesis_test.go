package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestGenesisState_Validate(t *testing.T) {
	val := sdk.ValAddress([]byte("validator-1_________")).String()
	entry := func(signed, member uint64) types.GenesisValidatorBlocks {
		return types.GenesisValidatorBlocks{ValidatorAddress: val, SignedBlocks: signed, MemberBlocks: member}
	}
	tests := []struct {
		name  string
		edit  func(gs *types.GenesisState)
		valid bool
	}{
		{"default", func(gs *types.GenesisState) {}, true},
		{"counters", func(gs *types.GenesisState) {
			gs.ValidatorBlocks, gs.EpochBlocks = []types.GenesisValidatorBlocks{entry(9, 10)}, 12
		}, true},
		{"invalid params", func(gs *types.GenesisState) { gs.Params.UnbondingPeriod = 0 }, false},
		{"invalid validator address", func(gs *types.GenesisState) {
			gs.ValidatorBlocks, gs.EpochBlocks = []types.GenesisValidatorBlocks{{ValidatorAddress: "nope", MemberBlocks: 1}}, 1
		}, false},
		{"duplicate validator", func(gs *types.GenesisState) {
			gs.ValidatorBlocks, gs.EpochBlocks = []types.GenesisValidatorBlocks{entry(1, 1), entry(1, 1)}, 1
		}, false},
		{"signed above member", func(gs *types.GenesisState) {
			gs.ValidatorBlocks, gs.EpochBlocks = []types.GenesisValidatorBlocks{entry(3, 2)}, 5
		}, false},
		{"member above epoch blocks", func(gs *types.GenesisState) {
			gs.ValidatorBlocks, gs.EpochBlocks = []types.GenesisValidatorBlocks{entry(1, 6)}, 5
		}, false},
		{"member zero", func(gs *types.GenesisState) {
			gs.ValidatorBlocks, gs.EpochBlocks = []types.GenesisValidatorBlocks{entry(0, 0)}, 5
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs := types.DefaultGenesis()
			tc.edit(gs)
			if tc.valid {
				require.NoError(t, gs.Validate())
			} else {
				require.Error(t, gs.Validate())
			}
		})
	}
}
