package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/xr/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
}

// ExchangeRateAuthorizations survive an export/import round-trip.
func TestGenesisRoundtripAuthorizations(t *testing.T) {
	f := initFixture(t)
	ts := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	op := sdk.AccAddress([]byte("xr_authz_operator___")).String()
	authz := types.ExchangeRateAuthorization{XrId: 1, Operator: op, Expiration: ts, MaxDeviationBps: 500}
	require.NoError(t, f.keeper.ExchangeRateAuthorizations.Set(f.ctx, collections.Join(authz.XrId, authz.Operator), authz))

	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Len(t, got.ExchangeRateAuthorizations, 1)

	f2 := initFixture(t)
	require.NoError(t, f2.keeper.InitGenesis(f2.ctx, *got))
	imported, err := f2.keeper.ExchangeRateAuthorizations.Get(f2.ctx, collections.Join(authz.XrId, authz.Operator))
	require.NoError(t, err)
	require.Equal(t, op, imported.Operator)
	require.Equal(t, uint32(500), imported.MaxDeviationBps)
	require.True(t, imported.Expiration.Equal(ts))
}
