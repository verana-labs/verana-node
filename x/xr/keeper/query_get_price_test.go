package keeper_test

import (
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func priceReq(amount string) *types.QueryGetPriceRequest {
	return &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
		Amount:         amount,
	}
}

// seedPrice creates uverana priced in USD at 500 / 10^2.
func seedPrice(t *testing.T, f *fixture, state bool) {
	t.Helper()
	msg := createMsg(t, f)
	msg.Rate = 500
	createRate(t, f, msg, state)
}

func TestGetPrice_SameAsset(t *testing.T) {
	f := initFixture(t)
	resp, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_COIN,
		QuoteAsset:     "uverana",
		Amount:         "1000",
	})
	require.NoError(t, err)
	require.Equal(t, "1000", resp.Price)
}

func TestGetPrice_ZeroAmount(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.GetPrice(f.ctx, priceReq("0"))
	require.NoError(t, err)
	require.Equal(t, "0", resp.Price)

	seedPrice(t, f, false)
	f.at(testNow.Add(time.Hour))
	resp, err = qs.GetPrice(f.ctx, priceReq("0"))
	require.NoError(t, err)
	require.Equal(t, "0", resp.Price)
}

func TestGetPrice_ZeroAmountInvalidAsset(t *testing.T) {
	f := initFixture(t)
	req := priceReq("0")
	req.QuoteAsset = "usd"
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, req)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "ISO-4217")
}

func TestGetPrice_TrustUnit(t *testing.T) {
	f := initFixture(t)
	req := priceReq("100")
	req.BaseAssetType, req.BaseAsset = cstypes.PricingAssetType_TU, "tu"
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, req)
	require.ErrorIs(t, err, types.ErrInvalidAssetType)
}

func TestGetPrice_NegativeAmount(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, priceReq("-1"))
	require.ErrorIs(t, err, types.ErrInvalidAmount)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetPrice_InvalidAmount(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, priceReq("1.5"))
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetPrice_ValidConversion(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	seedPrice(t, f, true)

	resp, err := qs.GetPrice(f.ctx, priceReq("100"))
	require.NoError(t, err)
	require.Equal(t, "500", resp.Price)

	resp, err = qs.GetPrice(f.ctx, priceReq("3"))
	require.NoError(t, err)
	require.Equal(t, "15", resp.Price)
}

func TestGetPrice_RoundsDown(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.Rate = 199
	createRate(t, f, msg, true)

	for amount, want := range map[int64]int64{1: 1, 3: 5, 100: 199} {
		price, err := f.keeper.GetPrice(f.ctx, cstypes.PricingAssetType_COIN, "uverana", cstypes.PricingAssetType_FIAT, "USD", math.NewInt(amount))
		require.NoError(t, err)
		require.Equal(t, math.NewInt(want), price, "amount %d", amount)
	}
}

func TestGetPrice_ExpiredRate(t *testing.T) {
	f := initFixture(t)
	seedPrice(t, f, true)
	f.at(testNow.Add(10 * time.Minute))
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, priceReq("100"))
	require.ErrorIs(t, err, types.ErrExchangeRateExpired)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGetPrice_NotActive(t *testing.T) {
	f := initFixture(t)
	seedPrice(t, f, false)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, priceReq("100"))
	require.ErrorIs(t, err, types.ErrExchangeRateNotActive)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGetPrice_NotFound(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, priceReq("100"))
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetPrice_Overflow(t *testing.T) {
	f := initFixture(t)
	seedPrice(t, f, true)
	_, err := keeper.NewQueryServerImpl(f.keeper).GetPrice(f.ctx, priceReq("1"+strings.Repeat("0", 76)))
	require.ErrorIs(t, err, types.ErrInvalidAmount)
}
