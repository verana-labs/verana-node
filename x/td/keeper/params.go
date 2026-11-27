package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/verana-labs/verana-node/x/td/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// MaxGasMsgTypeURLs returns the message types listed in max_gas_per_msg_type.
func (k Keeper) MaxGasMsgTypeURLs(ctx context.Context) ([]string, error) {
	entries := k.GetParams(ctx).MaxGasPerMsgType
	urls := make([]string, len(entries))
	for i, e := range entries {
		urls[i] = e.MsgTypeUrl
	}
	return urls, nil
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// GetPendingTuDecayRate returns the tu_decay_rate waiting for the next epoch boundary.
func (k Keeper) GetPendingTuDecayRate(ctx context.Context) (math.LegacyDec, bool, error) {
	rate, err := k.PendingTuDecayRate.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return math.LegacyDec{}, false, nil
	}
	if err != nil {
		return math.LegacyDec{}, false, err
	}
	return rate, true, nil
}

// ApplyPendingTuDecayRate writes the pending tu_decay_rate, step (4) of the epoch boundary.
func (k Keeper) ApplyPendingTuDecayRate(ctx context.Context) error {
	rate, found, err := k.GetPendingTuDecayRate(ctx)
	if err != nil || !found {
		return err
	}
	params := k.GetParams(ctx)
	params.TuDecayRate = rate
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}
	return k.PendingTuDecayRate.Remove(ctx)
}
