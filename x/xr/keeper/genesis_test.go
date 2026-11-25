package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func genesisWithFeeEntry() types.GenesisState {
	return types.GenesisState{
		Params: types.DefaultParams(),
		ExchangeRates: []types.ExchangeRate{{
			Id:               1,
			BaseAssetType:    cstypes.PricingAssetType_FIAT,
			BaseAsset:        "USD",
			QuoteAssetType:   cstypes.PricingAssetType_COIN,
			QuoteAsset:       "uvna",
			Rate:             1_000_000,
			ValidityDuration: 24 * time.Hour,
			Updated:          testNow,
			Expires:          testNow.Add(24 * time.Hour),
			State:            true,
		}, {
			Id:               2,
			BaseAssetType:    cstypes.PricingAssetType_FIAT,
			BaseAsset:        "EUR",
			QuoteAssetType:   cstypes.PricingAssetType_COIN,
			QuoteAsset:       "uvna",
			Rate:             1_100_000,
			ValidityDuration: time.Hour,
			Updated:          testNow,
			Expires:          testNow.Add(time.Hour),
		}},
		NextExchangeRateId: 2,
		ExchangeRateAuthorizations: []types.ExchangeRateAuthorization{
			{XrId: 1, Feeder: testAccount("feeder"), Expiration: testNow.Add(365 * 24 * time.Hour), MinInterval: time.Minute, MaxDeviationBps: 1000},
			{XrId: 2, Feeder: testAccount("feeder"), Expiration: testNow.Add(time.Hour)},
		},
	}
}

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

func TestGenesisRoundtrip(t *testing.T) {
	gs := genesisWithFeeEntry()
	f := initFixture(t)
	require.NoError(t, f.keeper.InitGenesis(f.ctx, gs))

	rate, scale, err := f.keeper.GenesisRate(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1_000_000), rate)
	require.Equal(t, uint32(0), scale)

	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, gs.ExchangeRates, got.ExchangeRates)
	require.Equal(t, gs.NextExchangeRateId, got.NextExchangeRateId)
	require.ElementsMatch(t, gs.ExchangeRateAuthorizations, got.ExchangeRateAuthorizations)

	f2 := initFixture(t)
	require.NoError(t, f2.keeper.InitGenesis(f2.ctx, *got))
	again, err := f2.keeper.ExportGenesis(f2.ctx)
	require.NoError(t, err)
	require.Equal(t, got, again)
}

func TestGenesis_FeeEntryAuthorizationNeedsMaxDeviation(t *testing.T) {
	gs := genesisWithFeeEntry()
	gs.ExchangeRateAuthorizations[0].MaxDeviationBps = 0
	require.NoError(t, gs.Validate())
	f := initFixture(t)
	require.ErrorContains(t, f.keeper.InitGenesis(f.ctx, gs), "max_deviation_bps")
}
