package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// maxWindowEndPopsPerBlock bounds the EndBlocker's work; the rest carries over.
const maxWindowEndPopsPerBlock = 1000

// EndBlock pops the due window-end keys and recomputes each affected VSOA's fee
// allowance once (MOD-DE-MSG-5-5). A key whose participant no longer has a
// record is discarded. A failed recompute is logged, never fatal.
func (k Keeper) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	var due []collections.Pair[time.Time, uint64]
	if err := k.WindowEndQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64]) (bool, error) {
		if key.K1().After(now) || len(due) >= maxWindowEndPopsPerBlock {
			return true, nil
		}
		due = append(due, key)
		return false, nil
	}); err != nil {
		return err
	}

	seen := map[uint64]bool{}
	var order []uint64
	for _, key := range due {
		if err := k.WindowEndQueue.Remove(ctx, key); err != nil {
			return err
		}
		vsoaID, err := k.VSOAByParticipant.Get(ctx, key.K2())
		if err != nil {
			continue
		}
		if !seen[vsoaID] {
			seen[vsoaID] = true
			order = append(order, vsoaID)
		}
	}

	for _, id := range order {
		vsoa, err := k.VSOperatorAuthorizations.Get(ctx, id)
		if err != nil {
			continue
		}
		if err := k.recomputeFeeAllowance(ctx, vsoa); err != nil {
			sdkCtx.Logger().Error("window-end recompute failed", "vsoa_id", id, "err", err)
		}
	}
	return nil
}
