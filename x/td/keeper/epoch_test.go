package keeper_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/td/keeper"
	"github.com/verana-labs/verana-node/x/td/types"
)

const day = 24 * time.Hour

var epochOrigin = time.Unix(1_700_000_000, 0).UTC()

func setupIndex(t *testing.T) (keeper.Keeper, sdk.Context, *keepertest.MockGenesisRateKeeper, *keepertest.MockDistributionKeeper) {
	k, ctx, xr, ds := keepertest.TrustdepositKeeperWithIndex(t)
	require.NoError(t, k.EpochOrigin.Set(ctx, epochOrigin.Unix()))
	return k, ctx, xr, ds
}

func blockAt(ctx sdk.Context, sinceOrigin time.Duration) sdk.Context {
	return ctx.WithBlockTime(epochOrigin.Add(sinceOrigin)).WithEventManager(sdk.NewEventManager())
}

func requirePeg(t *testing.T, ctx sdk.Context, k keeper.Keeper, want string) {
	t.Helper()
	peg, err := k.TuPegValue.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyMustNewDecFromStr(want), peg)
}

func requireLastClosed(t *testing.T, ctx sdk.Context, k keeper.Keeper, want uint64) {
	t.Helper()
	last, err := k.LastClosedEpoch.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, want, last)
}

func closeEpochEvents(ctx sdk.Context) []sdk.Event {
	var out []sdk.Event
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == types.EventTypeCloseEpoch {
			out = append(out, ev)
		}
	}
	return out
}

func TestEpochBoundary_NotReached(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	ctx = blockAt(ctx, day-time.Nanosecond)

	require.NoError(t, k.ProcessEpochBoundaries(ctx))

	requirePeg(t, ctx, k, "100")
	_, err := k.LastClosedEpoch.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
	require.Empty(t, ds.Epochs)
	require.Empty(t, closeEpochEvents(ctx))
}

func TestEpochBoundary_ClosesAtEnd(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	ctx = blockAt(ctx, day)

	require.NoError(t, k.ProcessEpochBoundaries(ctx))

	requirePeg(t, ctx, k, "99.9")
	requireLastClosed(t, ctx, k, 0)
	require.Equal(t, []uint64{0}, ds.Epochs)
	ev := lastEvent(t, ctx, types.EventTypeCloseEpoch)
	require.Equal(t, types.ModuleName, attrValue(ev, sdk.AttributeKeyModule))
	require.Equal(t, "0", attrValue(ev, "epoch"))
	require.Equal(t, "99.900000000000000000", attrValue(ev, "tu_peg_value"))
	require.Equal(t, "0.001000000000000000", attrValue(ev, "tu_decay_rate"))
	require.Empty(t, attrValue(ev, "pending_tu_decay_rate"))

	ctx = blockAt(ctx, 2*day-time.Second)
	require.NoError(t, k.ProcessEpochBoundaries(ctx))
	requirePeg(t, ctx, k, "99.9")
	require.Equal(t, []uint64{0}, ds.Epochs)
	require.Empty(t, closeEpochEvents(ctx))
}

func TestEpochBoundary_CatchUp(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	ctx = blockAt(ctx, 3*day+time.Hour)

	require.NoError(t, k.ProcessEpochBoundaries(ctx))

	requirePeg(t, ctx, k, "99.7002999")
	requireLastClosed(t, ctx, k, 2)
	require.Equal(t, []uint64{0, 1, 2}, ds.Epochs)
	evs := closeEpochEvents(ctx)
	require.Len(t, evs, 3)
	for i, ev := range evs {
		require.Equal(t, []string{"0", "1", "2"}[i], attrValue(ev, "epoch"))
	}
}

func TestEpochBoundary_PendingRate(t *testing.T) {
	k, ctx, _, _ := setupIndex(t)
	require.NoError(t, k.PendingTuDecayRate.Set(ctx, math.LegacyMustNewDecFromStr("0.002")))
	ctx = blockAt(ctx, 3*day)

	require.NoError(t, k.ProcessEpochBoundaries(ctx))

	// epoch 0 declines at the rate in force, the pending rate governs epochs 1 and 2
	requirePeg(t, ctx, k, "99.5007996")
	require.Equal(t, math.LegacyMustNewDecFromStr("0.002"), k.GetParams(ctx).TuDecayRate)
	_, found, err := k.GetPendingTuDecayRate(ctx)
	require.NoError(t, err)
	require.False(t, found)

	evs := closeEpochEvents(ctx)
	require.Len(t, evs, 3)
	require.Equal(t, "0.001000000000000000", attrValue(evs[0], "tu_decay_rate"))
	require.Equal(t, "0.002000000000000000", attrValue(evs[0], "pending_tu_decay_rate"))
	require.Equal(t, "0.002000000000000000", attrValue(evs[1], "tu_decay_rate"))
	require.Empty(t, attrValue(evs[1], "pending_tu_decay_rate"))
}

func TestEpochBoundary_PayoutBeforeDecay(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	require.NoError(t, k.PendingTuDecayRate.Set(ctx, math.LegacyMustNewDecFromStr("0.002")))
	var pegs, rates []string
	ds.OnPayout = func(c context.Context, _ uint64) error {
		peg, err := k.TuPegValue.Get(c)
		require.NoError(t, err)
		pegs = append(pegs, peg.String())
		rates = append(rates, k.GetParams(c).TuDecayRate.String())
		return nil
	}
	ctx = blockAt(ctx, 2*day)

	require.NoError(t, k.ProcessEpochBoundaries(ctx))

	require.Equal(t, []string{"100.000000000000000000", "99.900000000000000000"}, pegs)
	require.Equal(t, []string{"0.001000000000000000", "0.002000000000000000"}, rates)
}

func TestEpochBoundary_PayoutError(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	ds.OnPayout = func(context.Context, uint64) error { return errors.New("payout failed") }

	require.Error(t, k.ProcessEpochBoundaries(blockAt(ctx, day)))
}

func TestEpochBoundary_ResumesFromLastClosed(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	require.NoError(t, k.LastClosedEpoch.Set(ctx, 4))
	require.NoError(t, k.TuPegValue.Set(ctx, math.LegacyMustNewDecFromStr("99.5")))
	ctx = blockAt(ctx, 6*day+time.Minute)

	require.NoError(t, k.ProcessEpochBoundaries(ctx))

	require.Equal(t, []uint64{5}, ds.Epochs)
	requireLastClosed(t, ctx, k, 5)
	requirePeg(t, ctx, k, "99.4005")
}

func TestEpochBoundary_RoundsDown(t *testing.T) {
	k, ctx, _, _ := setupIndex(t)
	params := k.GetParams(ctx)
	params.TuDecayRate = math.LegacyMustNewDecFromStr("0.01")
	require.NoError(t, k.SetParams(ctx, params))
	require.NoError(t, k.TuPegValue.Set(ctx, math.LegacyMustNewDecFromStr("0.000000000000000003")))

	require.NoError(t, k.ProcessEpochBoundaries(blockAt(ctx, day)))

	requirePeg(t, ctx, k, "0.000000000000000002")
}

func TestEpochBoundary_EndBeyondDurationRange(t *testing.T) {
	k, ctx, _, ds := setupIndex(t)
	params := k.GetParams(ctx)
	params.EpochLength = 100 * 365 * day
	require.NoError(t, k.SetParams(ctx, params))

	// the end of epoch 2 lies past the time.Duration range from the origin
	require.NoError(t, k.ProcessEpochBoundaries(blockAt(ctx, 250*365*day)))

	require.Equal(t, []uint64{0, 1}, ds.Epochs)
	requireLastClosed(t, ctx, k, 1)
}

func TestEpochBoundary_WithoutDistribution(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	require.NoError(t, k.EpochOrigin.Set(ctx, epochOrigin.Unix()))

	require.NoError(t, k.ProcessEpochBoundaries(blockAt(ctx, day)))

	requirePeg(t, ctx, k, "99.9")
	requireLastClosed(t, ctx, k, 0)
}

func TestNativeDenomPrice(t *testing.T) {
	k, ctx, xr, _ := setupIndex(t)
	for _, tc := range []struct {
		rate  uint64
		scale uint32
		want  string
	}{
		{1_000_000, 0, "0.000001"},
		{3, 0, "0.333333333333333333"},
		{15, 1, "0.666666666666666666"},
		{1, 18, "1000000000000000000"},
	} {
		xr.Rate, xr.Scale = tc.rate, tc.scale
		price, err := k.NativeDenomPrice(ctx)
		require.NoError(t, err)
		require.Equal(t, math.LegacyMustNewDecFromStr(tc.want), price)
	}
}

func TestNativeDenomPrice_Unavailable(t *testing.T) {
	k, ctx, xr, _ := setupIndex(t)
	xr.Err = errors.New("exchange rate expired")

	_, err := k.NativeDenomPrice(ctx)
	require.ErrorIs(t, err, xr.Err)
}
