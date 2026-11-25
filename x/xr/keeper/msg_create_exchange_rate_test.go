package keeper_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func TestCreateExchangeRate_HappyPath(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	msg := createMsg(t, f)
	resp, err := ms.CreateExchangeRate(f.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.Id)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.ExchangeRateRecord{
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             100,
		RateScale:        2,
		ValidityDuration: 600,
		Updated:          testNow.Unix(),
		Expires:          testNow.Unix() + 600,
		State:            false,
	}, xr)

	attrs := eventAttrs(t, f, types.EventTypeCreateExchangeRate)
	require.Equal(t, "1", attrs[types.AttributeKeyXrID])
	require.Equal(t, "2023-11-14T22:13:20Z", attrs[types.AttributeKeyUpdated])
	require.Equal(t, "2023-11-14T22:23:20Z", attrs[types.AttributeKeyExpires])
	require.Contains(t, attrs["msg"], `"rate":"100"`)
}

func TestCreateExchangeRate_NoUpperValidityBound(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.ValidityDuration = 2 * 365 * 24 * time.Hour
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.NoError(t, err)
}

func TestCreateExchangeRate_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.Authority = testAccount("not_council")
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestCreateExchangeRate_InvalidAssetType(t *testing.T) {
	f := initFixture(t)
	for name, mutate := range map[string]func(*types.MsgCreateExchangeRate){
		"unspecified": func(m *types.MsgCreateExchangeRate) {
			m.BaseAssetType = cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED
		},
		"trust unit": func(m *types.MsgCreateExchangeRate) {
			m.QuoteAssetType = cstypes.PricingAssetType_TU
			m.QuoteAsset = "tu"
		},
	} {
		t.Run(name, func(t *testing.T) {
			msg := createMsg(t, f)
			mutate(msg)
			_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
			require.ErrorIs(t, err, types.ErrInvalidAssetType)
		})
	}
}

func TestCreateExchangeRate_FiatMustBeISO4217(t *testing.T) {
	f := initFixture(t)
	for _, code := range []string{"usd", "USDT"} {
		msg := createMsg(t, f)
		msg.QuoteAsset = code
		_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidAsset)
		require.Contains(t, err.Error(), "ISO-4217")
	}
}

func TestCreateExchangeRate_IdenticalPair(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.BaseAssetType, msg.BaseAsset = msg.QuoteAssetType, msg.QuoteAsset
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrIdenticalPair)
}

func TestCreateExchangeRate_DuplicatePair(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	_, err := ms.CreateExchangeRate(f.ctx, createMsg(t, f))
	require.NoError(t, err)

	msg := createMsg(t, f)
	msg.Rate = 200
	_, err = ms.CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrDuplicatePair)

	// the reverse direction is another pair
	msg.BaseAssetType, msg.BaseAsset, msg.QuoteAssetType, msg.QuoteAsset = msg.QuoteAssetType, msg.QuoteAsset, msg.BaseAssetType, msg.BaseAsset
	_, err = ms.CreateExchangeRate(f.ctx, msg)
	require.NoError(t, err)
}

func TestCreateExchangeRate_InvalidRate(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.Rate = 0
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidRate)
}

func TestCreateExchangeRate_InvalidRateScale(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.RateScale = 19
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidRateScale)

	msg.RateScale = 18
	_, err = keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.NoError(t, err)
}

func TestCreateExchangeRate_InvalidDuration(t *testing.T) {
	f := initFixture(t)
	for _, d := range []time.Duration{0, time.Minute - time.Second, time.Minute + time.Second/2, time.Duration(math.MaxUint32+1) * time.Second} {
		msg := createMsg(t, f)
		msg.ValidityDuration = d
		_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidDuration, d.String())
	}
}

func TestCreateExchangeRate_InvalidDenom(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.BaseAsset = "1uverana"
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidAsset)
}

func TestCreateExchangeRate_AssetNotOnChain(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.BaseAsset = "unknown"
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidRequest)

	msg = genesisPairMsg(t, f)
	msg.QuoteAsset = "unknown"
	_, err = keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidRequest)
}

func TestCreateExchangeRate_TokenfactoryDenom(t *testing.T) {
	f := initFixture(t)
	msg := createMsg(t, f)
	msg.BaseAsset = "factory/verana1abc/token"
	_, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidRequest)
}
