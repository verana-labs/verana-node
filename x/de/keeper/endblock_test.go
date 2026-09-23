package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [MOD-DE-MSG-5-5] window-end queue + EndBlocker.

func feegrantRecord(pid uint64, n int64) types.ParticipantAuthorizationRecord {
	return types.ParticipantAuthorizationRecord{
		ParticipantId: pid, MsgTypes: []string{mtCSPS}, WithFeegrant: true,
		FeeSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", n)),
	}
}

func queueLen(t *testing.T, f *fixture, ctx sdk.Context) int {
	t.Helper()
	n := 0
	require.NoError(t, f.keeper.WindowEndQueue.Walk(ctx, nil, func(_ collections.Pair[time.Time, uint64]) (bool, error) {
		n++
		return false, nil
	}))
	return n
}

func countEvents(ctx sdk.Context, typ string) int {
	n := 0
	for _, e := range ctx.EventManager().Events() {
		if e.Type == typ {
			n++
		}
	}
	return n
}

// A due key whose entry has expired drops its contribution; the last one
// revokes the allowance.
func TestEndBlock_ExpiredEntryRevokesAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	until := now.Add(time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until}

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 5)))
	has, _ := k.FeeGrants.Has(ctx, collections.Join(uint64(1), vsOp))
	require.True(t, has)
	require.Equal(t, 1, queueLen(t, f, ctx))

	// Not due yet: nothing happens.
	require.NoError(t, k.EndBlock(ctx.WithBlockTime(now.Add(30*time.Minute))))
	has, _ = k.FeeGrants.Has(ctx, collections.Join(uint64(1), vsOp))
	require.True(t, has)
	require.Equal(t, 1, queueLen(t, f, ctx))

	// Due: the entry is no longer active, the allowance goes, the key is popped.
	f.participants.views[10] = types.ParticipantView{EffectiveUntil: &until}
	require.NoError(t, k.EndBlock(ctx.WithBlockTime(until)))
	has, _ = k.FeeGrants.Has(ctx, collections.Join(uint64(1), vsOp))
	require.False(t, has)
	require.Equal(t, 0, queueLen(t, f, ctx))
}

// A due key with a sibling still alive only shrinks the sum.
func TestEndBlock_ExpiredEntryShrinksSum(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	until := ctx.BlockTime().Add(time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until}

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 5)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(11, 3)))
	fg, _ := k.FeeGrants.Get(ctx, collections.Join(uint64(1), vsOp))
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 8)), fg.SpendLimit)

	f.participants.views[10] = types.ParticipantView{EffectiveUntil: &until}
	require.NoError(t, k.EndBlock(ctx.WithBlockTime(until.Add(time.Second))))
	fg, _ = k.FeeGrants.Get(ctx, collections.Join(uint64(1), vsOp))
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 3)), fg.SpendLimit)
}

// A key left behind by a revoked record is discarded.
func TestEndBlock_OrphanKeyDiscarded(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	past := ctx.BlockTime().Add(-time.Minute)
	require.NoError(t, k.WindowEndQueue.Set(ctx, collections.Join(past, uint64(999))))

	require.NoError(t, k.EndBlock(ctx))
	require.Equal(t, 0, queueLen(t, f, ctx))
}

// An extended window is re-scheduled at its new end and keeps the allowance.
func TestEndBlock_ExtendedWindowRescheduled(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	until1 := ctx.BlockTime().Add(time.Hour)
	until2 := ctx.BlockTime().Add(3 * time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until1}
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 5)))

	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until2}
	require.NoError(t, k.EndBlock(ctx.WithBlockTime(until1)))

	has, _ := k.FeeGrants.Has(ctx, collections.Join(uint64(1), vsOp))
	require.True(t, has)
	old, _ := k.WindowEndQueue.Has(ctx, collections.Join(until1, uint64(10)))
	require.False(t, old)
	renewed, _ := k.WindowEndQueue.Has(ctx, collections.Join(until2, uint64(10)))
	require.True(t, renewed)
}

// Two due keys of one VSOA trigger a single recompute.
func TestEndBlock_RecomputesEachVSOAOnce(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	until := ctx.BlockTime().Add(time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until}
	f.participants.views[11] = types.ParticipantView{Active: true, EffectiveUntil: &until}
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 5)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(11, 3)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(12, 1)))

	due := ctx.WithBlockTime(until).WithEventManager(sdk.NewEventManager())
	f.participants.views[10] = types.ParticipantView{EffectiveUntil: &until}
	f.participants.views[11] = types.ParticipantView{EffectiveUntil: &until}
	require.NoError(t, k.EndBlock(due))

	require.Equal(t, 1, countEvents(due, types.EventTypeGrantFeeAllowance))
	fg, _ := k.FeeGrants.Get(ctx, collections.Join(uint64(1), vsOp))
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 1)), fg.SpendLimit)
	require.Equal(t, 0, queueLen(t, f, ctx))
}
