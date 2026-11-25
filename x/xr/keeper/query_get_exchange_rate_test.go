package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func requireNotFound(t *testing.T, err error) {
	t.Helper()
	require.Equal(t, codes.NotFound, status.Code(err), "%v", err)
}

func TestGetExchangeRate_ById(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)

	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, types.ExchangeRate{
		Id:               id,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             100,
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
		Updated:          testNow,
		Expires:          testNow.Add(10 * time.Minute),
		State:            true,
	}, resp.ExchangeRate)
	require.Empty(t, resp.Authorizations)
}

func TestGetExchangeRate_ByPair(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	createRate(t, f, genesisPairMsg(t, f), true)
	id := createRate(t, f, createMsg(t, f), true)

	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
	})
	require.NoError(t, err)
	require.Equal(t, id, resp.ExchangeRate.Id)
}

func TestGetExchangeRate_NotFound(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	createRate(t, f, createMsg(t, f), true)

	_, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: 999})
	requireNotFound(t, err)

	_, err = qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "EUR",
	})
	requireNotFound(t, err)
}

func TestGetExchangeRate_IncompletePair(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{BaseAsset: "uverana"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetExchangeRate_StateFilter(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	disabled := createRate(t, f, createMsg(t, f), false)
	enabled := createRate(t, f, genesisPairMsg(t, f), true)

	for id, state := range map[uint64]types.StateFilter{
		disabled: types.StateFilter_STATE_FILTER_ACTIVE,
		enabled:  types.StateFilter_STATE_FILTER_INACTIVE,
	} {
		_, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id, State: state})
		requireNotFound(t, err)
	}
	for id, state := range map[uint64]types.StateFilter{
		disabled: types.StateFilter_STATE_FILTER_INACTIVE,
		enabled:  types.StateFilter_STATE_FILTER_ACTIVE,
	} {
		resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id, State: state})
		require.NoError(t, err)
		require.Equal(t, id, resp.ExchangeRate.Id)
	}
}

func TestGetExchangeRate_ExpiresFilter(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)

	before := testNow.Add(10*time.Minute - time.Second)
	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id, Expires: &before})
	require.NoError(t, err)
	require.Equal(t, id, resp.ExchangeRate.Id)

	at := testNow.Add(10 * time.Minute)
	_, err = qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id, Expires: &at})
	requireNotFound(t, err)
}

func TestGetExchangeRate_IncludesAuthorizations(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)
	other := createRate(t, f, genesisPairMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, time.Minute, 500)
	grantAuthz(t, f, other, testAccount("other_feeder"), 0, 500)

	resp, err := keeper.NewQueryServerImpl(f.keeper).GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, []types.ExchangeRateAuthorization{{
		XrId:            id,
		Feeder:          feeder,
		Expiration:      testNow.Add(time.Hour),
		MinInterval:     time.Minute,
		MaxDeviationBps: 500,
	}}, resp.Authorizations)
}

func TestGetExchangeRate_NilRequest(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetExchangeRate(f.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
