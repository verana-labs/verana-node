package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

// helper to create an exchange rate and return its id
func createTestExchangeRate(t *testing.T, f *fixture, ms types.MsgServer, authorityStr string) uint64 {
	t.Helper()
	resp, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)
	return resp.Id
}

func TestSetExchangeRateState_HappyPath_Enable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// [MOD-XR-MSG-1-3] created disabled.
	id := createTestExchangeRate(t, f, ms, authorityStr)
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)

	// [MOD-XR-MSG-3-3] set state to the supplied value (enable).
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     true,
	})
	require.NoError(t, err)

	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)

	// Re-submitting the same target value is idempotent (not a toggle).
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     true,
	})
	require.NoError(t, err)

	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)
}

func TestSetExchangeRateState_HappyPath_Disable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	id := createTestExchangeRate(t, f, ms, authorityStr)

	// Enable, then explicitly disable; state tracks the supplied value.
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     true,
	})
	require.NoError(t, err)

	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     false,
	})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)
}

func TestSetExchangeRateState_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	id := createTestExchangeRate(t, f, ms, authorityStr)

	nonGovAddr := sdk.AccAddress([]byte("not_gov_authority___")).String()
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: nonGovAddr,
		Id:        id,
		State:     true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected gov account as only signer")
}

func TestSetExchangeRateState_NotFound(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        999,
		State:     true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exchange rate not found")
}
