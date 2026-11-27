package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestGenesis(t *testing.T) {
	f := initFixture(t)
	p := types.DefaultParams()
	p.CouncilBudget = math.LegacyNewDec(42)
	a := sdk.ValAddress([]byte("validator-a_________")).String()
	b := sdk.ValAddress([]byte("validator-b_________")).String()
	gs := types.GenesisState{
		Params: p,
		ValidatorBlocks: []types.GenesisValidatorBlocks{
			{ValidatorAddress: a, SignedBlocks: 9, MemberBlocks: 10},
			{ValidatorAddress: b, SignedBlocks: 0, MemberBlocks: 4},
		},
		EpochBlocks: 10,
	}

	require.NoError(t, f.keeper.InitGenesis(f.ctx, gs))
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, gs, *got)
}

func TestGenesis_Default(t *testing.T) {
	f := initFixture(t)
	gs := types.DefaultGenesis()
	require.NoError(t, f.keeper.InitGenesis(f.ctx, *gs))
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, gs, got)
}

func TestGenesis_FoundationAccountBlocked(t *testing.T) {
	f := initFixture(t)
	gs := types.DefaultGenesis()
	gs.Params.FoundationAccount = sdk.AccAddress(authtypes.NewModuleAddress("bonded_tokens_pool")).String()
	f.bank.blocked[gs.Params.FoundationAccount] = true
	require.ErrorIs(t, f.keeper.InitGenesis(f.ctx, *gs), types.ErrInvalidParams)
}
