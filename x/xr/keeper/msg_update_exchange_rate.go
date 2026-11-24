package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// UpdateExchangeRate implements [MOD-XR-MSG-2] Update Exchange Rate.
func (ms msgServer) UpdateExchangeRate(ctx context.Context, msg *types.MsgUpdateExchangeRate) (*types.MsgUpdateExchangeRateResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	feeder, err := ms.addressCodec.StringToBytes(msg.Feeder)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid feeder address")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime().Unix()

	xr, err := ms.ExchangeRates.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.Id)
	}
	if !xr.State {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotActive, "exchange rate with id %d is not active", msg.Id)
	}

	authz, err := ms.ExchangeRateAuthorizations.Get(ctx, collections.Join(msg.Id, sdk.AccAddress(feeder)))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrAuthorizationNotFound, "feeder %s is not authorized to update exchange rate %d", msg.Feeder, msg.Id)
	}
	if now >= authz.Expiration {
		return nil, types.ErrAuthorizationExpired
	}
	if authz.MinInterval > 0 && now-xr.Updated < int64(authz.MinInterval) {
		return nil, errorsmod.Wrapf(types.ErrUpdateTooSoon, "min_interval %ds not elapsed since last update", authz.MinInterval)
	}
	if authz.MaxDeviationBps > 0 && exceedsDeviation(xr.Rate, msg.Rate, authz.MaxDeviationBps) {
		return nil, errorsmod.Wrapf(types.ErrRateDeviationExceeded, "rate change exceeds max_deviation_bps %d", authz.MaxDeviationBps)
	}

	xr.Rate = msg.Rate
	xr.Expires = now + int64(xr.ValidityDuration)
	xr.Updated = now
	if err := ms.ExchangeRates.Set(ctx, msg.Id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store updated exchange rate")
	}

	events.New(types.ModuleName, types.EventTypeUpdateExchangeRate).
		Uint(types.AttributeKeyXrID, msg.Id).
		Time(types.AttributeKeyUpdated, time.Unix(xr.Updated, 0)).
		Time(types.AttributeKeyExpires, time.Unix(xr.Expires, 0)).
		Emit(sdkCtx)

	return &types.MsgUpdateExchangeRateResponse{}, nil
}

// exceedsDeviation reports |next - current| * 10000 > bps * current.
func exceedsDeviation(current, next uint64, bps uint32) bool {
	diff := next - current
	if next < current {
		diff = current - next
	}
	return math.NewIntFromUint64(diff).MulRaw(10000).GT(math.NewIntFromUint64(current).MulRaw(int64(bps)))
}
