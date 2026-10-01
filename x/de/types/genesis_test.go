package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-node/x/de/types"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "valid genesis state",
			genState: &types.GenesisState{Params: types.DefaultParams()},
			valid:    true,
		},
		{
			desc:     "zero fee period is invalid",
			genState: &types.GenesisState{},
			valid:    false,
		},
		{
			desc: "queue entry without participant is invalid",
			genState: &types.GenesisState{
				Params:         types.DefaultParams(),
				WindowEndQueue: []types.WindowEndQueueEntry{{ParticipantId: 0}},
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
