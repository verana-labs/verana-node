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

func TestGenesis_Sponsorships(t *testing.T) {
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Sponsorships: []types.Sponsorship{
			{SponsorCorporationId: 1, SponsoredCorporationId: 2, CoversFees: true, Expiration: &exp, Created: exp.AddDate(-1, 0, 0)},
			{SponsorCorporationId: 3, SponsoredCorporationId: 1, SpendLimit: uvna(10), RemainingSpend: uvna(4), Created: exp.AddDate(-2, 0, 0)},
		},
	}

	f := initFixture(t)
	require.NoError(t, f.keeper.InitGenesis(f.ctx, genesisState))
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, genesisState.Sponsorships, got.Sponsorships)
}

func TestGenesis_RebuildsDelegateIndex(t *testing.T) {
	vsOp := acc("vsop_genesis________")
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		VsOperatorAuthorizations: []types.VSOperatorAuthorization{{
			Id: 1, CorporationId: 4, VsOperator: vsOp,
			Records: []types.ParticipantAuthorizationRecord{{ParticipantId: 9, MsgTypes: []types.MsgType{mtValidated}}},
		}},
		VsoaSeq: 1,
	}

	f := initFixture(t)
	require.NoError(t, f.keeper.InitGenesis(f.ctx, genesisState))
	corp, err := f.keeper.VSOACorpByOperator.Get(f.ctx, vsOp)
	require.NoError(t, err)
	require.Equal(t, uint64(4), corp)
}

func TestGenesis_DelegateAuthorizations(t *testing.T) {
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	period := time.Hour
	das := []types.DelegateAuthorization{
		{CorporationId: 1, Delegate: acc("delegate_b__________"), MsgTypes: []types.MsgType{mtEcosystem}, AcceptedCgfVersion: 2},
		{CorporationId: 2, Delegate: acc("delegate_a__________"), MsgTypes: []types.MsgType{mtSchema},
			SpendLimit: uvna(5), RemainingSpend: uvna(3), Expiration: &exp, Period: &period},
	}

	f := initFixture(t)
	require.NoError(t, f.keeper.InitGenesis(f.ctx, types.GenesisState{Params: types.DefaultParams(), DelegateAuthorizations: das}))
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, das, got.DelegateAuthorizations)
}
