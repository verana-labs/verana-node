package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	feegrant "cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [MOD-DE-MSG-5-5] The aggregate allowance is re-granted on every recompute;
// what was already spent in the current period must not come back.

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

// A sync mid-period keeps the spent amount and the reset time.
func TestRecomputeFeeAllowance_CarriesSpentBudget(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	reset := now.Add(types.DefaultVsOperatorFeePeriod)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 10)))

	spendFee(t, f, ctx, 1, vsOp, uvna(7))
	require.Equal(t, uvna(3), periodic(t, f, ctx, 1, vsOp).PeriodCanSpend)

	later := ctx.WithBlockTime(now.Add(10 * time.Minute))
	require.NoError(t, k.SyncVSOperatorAuthorization(later, 10))

	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, uvna(10), pa.PeriodSpendLimit)
	require.Equal(t, uvna(3), pa.PeriodCanSpend, "spent 7 stays spent")
	require.True(t, pa.PeriodReset.Equal(reset), "reset not pushed out")
	fg, err := k.FeeGrants.Get(ctx, collections.Join(uint64(1), vsOp))
	require.NoError(t, err)
	require.Equal(t, uvna(3), fg.RemainingSpend)
	require.True(t, fg.Expiration.Equal(reset))
}

// A record added mid-period raises the limit by its share; the spent amount
// stays spent.
func TestRecomputeFeeAllowance_RaisedLimitKeepsSpent(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 10)))
	spendFee(t, f, ctx, 1, vsOp, uvna(7))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(11, 5)))

	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, uvna(15), pa.PeriodSpendLimit)
	require.Equal(t, uvna(8), pa.PeriodCanSpend)
}

// A record removed mid-period lowers the limit; a budget already overdrawn
// against the new limit floors at zero and stays blocked until the reset.
func TestRecomputeFeeAllowance_LoweredLimitFloorsAtZero(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 6)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(11, 4)))
	spendFee(t, f, ctx, 1, vsOp, uvna(7))

	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 10))

	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, uvna(4), pa.PeriodSpendLimit)
	require.True(t, pa.PeriodCanSpend.IsZero())
	_, err := pa.Accept(ctx, uvna(1), nil)
	require.ErrorIs(t, err, feegrant.ErrFeeLimitExceeded)
}

// Once the period has elapsed the recompute hands out a fresh budget, exactly
// as the allowance itself would on the next draw.
func TestRecomputeFeeAllowance_FreshBudgetAfterPeriod(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 10)))
	spendFee(t, f, ctx, 1, vsOp, uvna(7))

	later := ctx.WithBlockTime(now.Add(25 * time.Hour))
	require.NoError(t, k.SyncVSOperatorAuthorization(later, 10))

	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, uvna(10), pa.PeriodCanSpend)
	require.True(t, pa.PeriodReset.Equal(now.Add(25*time.Hour).Add(types.DefaultVsOperatorFeePeriod)))
}

// A stale window-end pop (the window was extended) must not refill the budget.
func TestEndBlock_StalePopKeepsBudget(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	until1 := now.Add(time.Hour)
	until2 := now.Add(3 * time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until1}
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 10)))
	spendFee(t, f, ctx, 1, vsOp, uvna(7))

	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until2}
	require.NoError(t, k.EndBlock(ctx.WithBlockTime(until1)))

	pa := periodic(t, f, ctx, 1, vsOp)
	require.Equal(t, uvna(3), pa.PeriodCanSpend)
	require.True(t, pa.PeriodReset.Equal(now.Add(types.DefaultVsOperatorFeePeriod)))
}

// [MOD-DE-MSG-1-4] The operator path is untouched: an explicit re-grant seeds
// a fresh budget.
func TestGrantFeeAllowance_OperatorPathFreshBudget(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")
	period := time.Hour
	reset := ctx.BlockTime().Add(period)
	require.NoError(t, k.GrantFeeAllowance(ctx, 7, grantee, []string{mtEcosystem}, &reset, uvna(10), &period))
	spendFee(t, f, ctx, 7, grantee, uvna(7))

	reset2 := ctx.BlockTime().Add(2 * period)
	require.NoError(t, k.GrantFeeAllowance(ctx, 7, grantee, []string{mtEcosystem}, &reset2, uvna(10), &period))

	pa := periodic(t, f, ctx, 7, grantee)
	require.Equal(t, uvna(10), pa.PeriodCanSpend)
	require.True(t, pa.PeriodReset.Equal(reset2))
}
