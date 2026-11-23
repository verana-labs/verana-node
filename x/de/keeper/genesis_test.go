package keeper_test

import (
	"testing"
	"time"

	"github.com/verana-labs/verana-node/x/de/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	until := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	genesisState := types.GenesisState{
		Params:         types.DefaultParams(),
		WindowEndQueue: []types.WindowEndQueueEntry{{WindowEnd: until, ParticipantId: 7}},
	}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.Len(t, got.WindowEndQueue, 1)
	require.True(t, got.WindowEndQueue[0].WindowEnd.Equal(until))
	require.Equal(t, uint64(7), got.WindowEndQueue[0].ParticipantId)
}

func TestGenesis_RebuildsOperatorIndex(t *testing.T) {
	vsOp := acc("vsop_genesis________")
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		VsOperatorAuthorizations: []types.VSOperatorAuthorization{{
			Id: 1, CorporationId: 4, VsOperator: vsOp,
			Records: []types.ParticipantAuthorizationRecord{{ParticipantId: 9, MsgTypes: []string{mtValidated}}},
		}},
		VsoaSeq: 1,
	}

	f := initFixture(t)
	require.NoError(t, f.keeper.InitGenesis(f.ctx, genesisState))
	corp, err := f.keeper.VSOACorpByOperator.Get(f.ctx, vsOp)
	require.NoError(t, err)
	require.Equal(t, uint64(4), corp)
}
