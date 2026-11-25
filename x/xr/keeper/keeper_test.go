package keeper_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/keeper"
	module "github.com/verana-labs/verana-node/x/xr/module"
	"github.com/verana-labs/verana-node/x/xr/types"
)

var testNow = time.Unix(1_700_000_000, 0).UTC()

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
}

// mockBankKeeper treats every denom but "unknown" as existing on-chain.
type mockBankKeeper struct{}

func (mockBankKeeper) SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}
func (mockBankKeeper) HasSupply(_ context.Context, denom string) bool { return denom != "unknown" }

type mockTrustDepositKeeper struct{}

func (mockTrustDepositKeeper) MainFiatCurrency(context.Context) string { return "USD" }

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx.WithBlockTime(testNow)

	authority := authtypes.NewModuleAddress("council")

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		mockBankKeeper{},
	)
	k.SetTrustDepositKeeper(mockTrustDepositKeeper{})

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
	}
}

func (f *fixture) authority(t *testing.T) string {
	t.Helper()
	s, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	return s
}

// at moves the block time.
func (f *fixture) at(blockTime time.Time) {
	f.ctx = sdk.UnwrapSDKContext(f.ctx).WithBlockTime(blockTime)
}

func testAccount(name string) string {
	return sdk.AccAddress(fmt.Appendf(nil, "%-20s", name)).String()
}

// createMsg prices uverana in USD, which is not the genesis pair.
func createMsg(t *testing.T, f *fixture) *types.MsgCreateExchangeRate {
	return &types.MsgCreateExchangeRate{
		Authority:        f.authority(t),
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             100,
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	}
}

// genesisPairMsg prices the main fiat currency in the native denom.
func genesisPairMsg(t *testing.T, f *fixture) *types.MsgCreateExchangeRate {
	return &types.MsgCreateExchangeRate{
		Authority:        f.authority(t),
		BaseAssetType:    cstypes.PricingAssetType_FIAT,
		BaseAsset:        "USD",
		QuoteAssetType:   cstypes.PricingAssetType_COIN,
		QuoteAsset:       "uvna",
		Rate:             1_000_000,
		RateScale:        0,
		ValidityDuration: time.Hour,
	}
}

// createRate creates the entry of msg and sets its state.
func createRate(t *testing.T, f *fixture, msg *types.MsgCreateExchangeRate, state bool) uint64 {
	t.Helper()
	resp, err := keeper.NewMsgServerImpl(f.keeper).CreateExchangeRate(f.ctx, msg)
	require.NoError(t, err)
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	xr.State = state
	require.NoError(t, f.keeper.ExchangeRates.Set(f.ctx, resp.Id, xr))
	return resp.Id
}

// grantAuthz grants feeder on id; zero minInterval or maxDevBps is unset.
func grantAuthz(t *testing.T, f *fixture, id uint64, feeder string, minInterval time.Duration, maxDevBps uint32) {
	t.Helper()
	exp := sdk.UnwrapSDKContext(f.ctx).BlockTime().Add(time.Hour)
	msg := &types.MsgGrantExchangeRateAuthorization{
		Authority:       f.authority(t),
		XrId:            id,
		Feeder:          feeder,
		Expiration:      &exp,
		MaxDeviationBps: maxDevBps,
	}
	if minInterval > 0 {
		msg.MinInterval = &minInterval
	}
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
	require.NoError(t, err)
}

// eventAttrs returns the attributes of the last event of eventType.
func eventAttrs(t *testing.T, f *fixture, eventType string) map[string]string {
	t.Helper()
	var attrs map[string]string
	for _, ev := range sdk.UnwrapSDKContext(f.ctx).EventManager().Events() {
		if ev.Type == eventType {
			attrs = map[string]string{}
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
		}
	}
	require.NotNil(t, attrs, "no %s event", eventType)
	return attrs
}

func TestGenesisRate(t *testing.T) {
	f := initFixture(t)

	_, _, err := f.keeper.GenesisRate(f.ctx)
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)
	_, _, err = f.keeper.LastGenesisRate(f.ctx)
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)

	createRate(t, f, createMsg(t, f), true)
	id := createRate(t, f, genesisPairMsg(t, f), false)

	_, _, err = f.keeper.GenesisRate(f.ctx)
	require.ErrorIs(t, err, types.ErrExchangeRateNotActive)
	rate, scale, err := f.keeper.LastGenesisRate(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1_000_000), rate)
	require.Equal(t, uint32(0), scale)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	xr.State = true
	require.NoError(t, f.keeper.ExchangeRates.Set(f.ctx, id, xr))
	rate, scale, err = f.keeper.GenesisRate(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1_000_000), rate)
	require.Equal(t, uint32(0), scale)
}

func TestGenesisRate_Expired(t *testing.T) {
	f := initFixture(t)
	createRate(t, f, genesisPairMsg(t, f), true)

	f.at(testNow.Add(time.Hour))
	_, _, err := f.keeper.GenesisRate(f.ctx)
	require.ErrorIs(t, err, types.ErrExchangeRateExpired)

	rate, _, err := f.keeper.LastGenesisRate(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1_000_000), rate)
}
