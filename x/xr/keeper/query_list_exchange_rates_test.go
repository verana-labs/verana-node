package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func seedMultipleExchangeRates(t *testing.T, f *fixture) {
	t.Helper()
	ms := keeper.NewMsgServerImpl(f.keeper)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// Rate 1: COIN/uverana -> FIAT/USD (left disabled per [MOD-XR-MSG-1-3])
	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "500",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	// Rate 2: COIN/uverana -> FIAT/EUR (left disabled per [MOD-XR-MSG-1-3])
	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "EUR",
		Rate:             "450",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	// Rate 3: TU/TU -> FIAT/USD (explicitly enabled for the active-filter test)
	resp3, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_TU,
		BaseAsset:        "tu",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        resp3.Id,
		State:     true,
	})
	require.NoError(t, err)
}

func TestListExchangeRates_All(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	seedMultipleExchangeRates(t, f)

	resp, err := qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 3)
}

func TestListExchangeRates_FilterByAssetType(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	seedMultipleExchangeRates(t, f)

	// Filter by base_asset_type = TU
	resp, err := qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{
		BaseAssetType: cstypes.PricingAssetType_TU,
	})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 1)
	require.Equal(t, "tu", resp.ExchangeRates[0].BaseAsset)

	// Filter by quote_asset = EUR
	resp, err = qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{
		QuoteAsset: "EUR",
	})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 1)
	require.Equal(t, "EUR", resp.ExchangeRates[0].QuoteAsset)

	// Filter by state = active (only rate 3 is active)
	resp, err = qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{
		State: types.StateFilter_STATE_FILTER_ACTIVE,
	})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 1)
	require.Equal(t, "tu", resp.ExchangeRates[0].BaseAsset)
}

func TestListExchangeRates_EmptyResult(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.ExchangeRates)
}
