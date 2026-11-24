package keeper

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/td/types"
)

// ProcessEpochBoundaries closes, in sequence, every epoch that ended at or before the block time.
func (k Keeper) ProcessEpochBoundaries(ctx sdk.Context) error {
	origin, err := k.EpochOrigin.Get(ctx)
	if err != nil {
		return err
	}
	next := uint64(0)
	last, err := k.LastClosedEpoch.Get(ctx)
	switch {
	case err == nil:
		next = last + 1
	case !errors.Is(err, collections.ErrNotFound):
		return err
	}
	start, length := time.Unix(origin, 0), k.GetParams(ctx).EpochLength
	for ; !start.Add(time.Duration(next+1) * length).After(ctx.BlockTime()); next++ {
		if err := k.closeEpoch(ctx, next); err != nil {
			return err
		}
	}
	return nil
}

// closeEpoch performs the boundary steps of the Trust Unit Index for epoch.
func (k Keeper) closeEpoch(ctx sdk.Context, epoch uint64) error {
	if k.distributionKeeper != nil {
		if err := k.distributionKeeper.PayoutEpoch(ctx, epoch); err != nil {
			return err
		}
	}
	rate := k.GetParams(ctx).TuDecayRate
	peg, err := k.TuPegValue.Get(ctx)
	if err != nil {
		return err
	}
	peg = peg.MulTruncate(math.LegacyOneDec().Sub(rate))
	if err := k.TuPegValue.Set(ctx, peg); err != nil {
		return err
	}
	applied, err := k.applyPendingTuDecayRate(ctx)
	if err != nil {
		return err
	}
	if err := k.LastClosedEpoch.Set(ctx, epoch); err != nil {
		return err
	}
	events.New(types.ModuleName, types.EventTypeCloseEpoch).
		Uint(types.AttributeKeyEpoch, epoch).
		Dec(types.AttributeKeyTuPegValue, peg).
		Dec(types.AttributeKeyTuDecayRate, rate).
		Dec(types.AttributeKeyPendingTuDecayRate, applied).
		Emit(ctx)
	return nil
}

// NativeDenomPrice returns P(now), the price of one base unit of native denom in minor units of the
// main fiat currency, from the genesis exchange rate entry, rounded down to the ledger precision.
func (k Keeper) NativeDenomPrice(ctx context.Context) (math.LegacyDec, error) {
	rate, scale, err := k.xrKeeper.GetGenesisRate(ctx, k.GetParams(ctx).MainFiatCurrency, types.BondDenom)
	if err != nil {
		return math.LegacyDec{}, err
	}
	return math.LegacyNewDecFromInt(math.NewIntWithDecimal(1, int(scale))).
		QuoTruncate(math.LegacyNewDecFromInt(math.NewIntFromUint64(rate))), nil
}
