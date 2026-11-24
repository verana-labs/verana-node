package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [MOD-DE-MSG-5-5] The aggregate allowance is re-granted on every recompute,
// each time with a fresh period budget.

func uvna(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("uvna", n)) }

// periodic returns the PeriodicAllowance realized for (corp, grantee).
func periodic(t *testing.T, f *fixture, ctx sdk.Context, corpID uint64, grantee string) *feegrant.PeriodicAllowance {
	t.Helper()
	granter, _ := sdk.AccAddressFromBech32(corpPolicyAddr(corpID))
	granteeAddr, _ := sdk.AccAddressFromBech32(grantee)
	a, err := f.feegrants.GetAllowance(ctx, granter, granteeAddr)
	require.NoError(t, err)
	wrapped, ok := a.(*feegrant.AllowedMsgAllowance)
	require.True(t, ok)
	inner, err := wrapped.GetAllowance()
	require.NoError(t, err)
	pa, ok := inner.(*feegrant.PeriodicAllowance)
	require.True(t, ok)
	return pa
}

// spendFee draws fee from the allowance the way UseGrantedFees does: the
// SDK's own Accept, then the mutated allowance is written back.
func spendFee(t *testing.T, f *fixture, ctx sdk.Context, corpID uint64, grantee string, fee sdk.Coins) {
	t.Helper()
	granter, _ := sdk.AccAddressFromBech32(corpPolicyAddr(corpID))
	granteeAddr, _ := sdk.AccAddressFromBech32(grantee)
	a, err := f.feegrants.GetAllowance(ctx, granter, granteeAddr)
	require.NoError(t, err)
	wrapped := a.(*feegrant.AllowedMsgAllowance)
	inner, err := wrapped.GetAllowance()
	require.NoError(t, err)
	remove, err := inner.Accept(ctx, fee, nil)
	require.NoError(t, err)
	require.False(t, remove)
	require.NoError(t, wrapped.SetAllowance(inner))
}

func TestRecomputeFeeAllowance_RealizesPeriodicAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	require.NoError(t, f.keeper.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 10)))

	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, types.DefaultVsOperatorFeePeriod, pa.Period)
	require.Equal(t, uvna(10), pa.PeriodSpendLimit)
	require.Equal(t, uvna(10), pa.PeriodCanSpend)
	require.True(t, pa.PeriodReset.Equal(now.Add(types.DefaultVsOperatorFeePeriod)))
	require.Nil(t, pa.Basic.SpendLimit)
	require.Nil(t, pa.Basic.Expiration)
}

// A recompute mid-period refreshes the budget and starts a new period.
func TestRecomputeFeeAllowance_ResetsBudget(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 10)))
	spendFee(t, f, ctx, 1, vsOp, uvna(7))

	later := ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Minute))
	require.NoError(t, k.GrantVSOperatorAuthorization(later, 1, vsOp, feegrantRecord(11, 5)))

	reset := later.BlockTime().Add(types.DefaultVsOperatorFeePeriod)
	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, uvna(15), pa.PeriodSpendLimit)
	require.Equal(t, uvna(15), pa.PeriodCanSpend)
	require.True(t, pa.PeriodReset.Equal(reset))
	fg, err := k.FeeGrants.Get(ctx, collections.Join(uint64(1), vsOp))
	require.NoError(t, err)
	require.Equal(t, uvna(15), fg.RemainingSpend)
	require.True(t, fg.Expiration.Equal(reset))
}

// [MOD-DE-MSG-1-4] A re-grant seeds the period budget with the spend limit.
func TestGrantFeeAllowance_RegrantResetsBudget(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")
	period := time.Hour
	reset := ctx.BlockTime().Add(period)
	require.NoError(t, k.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &reset, uvna(10), &period))
	spendFee(t, f, ctx, 7, grantee, uvna(7))

	reset2 := ctx.BlockTime().Add(2 * period)
	require.NoError(t, k.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &reset2, uvna(10), &period))

	pa := periodic(t, f, ctx, 7, grantee)
	require.Equal(t, uvna(10), pa.PeriodCanSpend)
	require.True(t, pa.PeriodReset.Equal(reset2))
}
