package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/xr/types"
)

// UpdateExchangeRate implements [MOD-XR-MSG-2] Update Exchange Rate.
func (ms msgServer) UpdateExchangeRate(ctx context.Context, msg *types.MsgUpdateExchangeRate) (*types.MsgUpdateExchangeRateResponse, error) {
	// Validate basic fields
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// Load ExchangeRate by id
	xr, err := ms.ExchangeRates.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.Id)
	}

	// [MOD-XR-MSG-2] Authorization: an ExchangeRateAuthorization (xr_id, operator)
	// MUST exist and MUST NOT be expired.
	authzKey := collections.Join(msg.Id, msg.Operator)
	authz, err := ms.ExchangeRateAuthorizations.Get(ctx, authzKey)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrAuthorizationNotFound, "operator %s is not authorized to update exchange rate %d", msg.Operator, msg.Id)
	}
	if !authz.Expiration.After(now) {
		return nil, types.ErrAuthorizationExpired
	}

	// [MOD-XR-MSG-2-2-1] xr.state MUST be enabled (the only state gate; an expired
	// rate is still refreshable by its operator).
	if !xr.State {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotActive, "exchange rate with id %d is not active", msg.Id)
	}

	// Anti-spam: if min_interval is set, reject updates that arrive too soon
	// after the last successful update.
	if authz.MinInterval > 0 && now.Sub(xr.Updated) < authz.MinInterval {
		return nil, errorsmod.Wrapf(types.ErrUpdateTooSoon, "min_interval %s not elapsed since last update", authz.MinInterval)
	}

	// Circuit breaker: if max_deviation_bps is set, reject changes whose relative
	// deviation exceeds max_deviation_bps/10000. Computed as
	// |new - old| / old <= bps/10000  ⟺  |new - old| * 10000 <= bps * old.
	if authz.MaxDeviationBps > 0 {
		newRate, _ := math.NewIntFromString(msg.Rate) // validated in ValidateBasic
		oldRate, ok := math.NewIntFromString(xr.Rate)
		if !ok || !oldRate.IsPositive() {
			return nil, errorsmod.Wrapf(types.ErrInvalidRate, "stored rate %q is invalid", xr.Rate)
		}
		lhs := newRate.Sub(oldRate).Abs().Mul(math.NewInt(10000))
		rhs := oldRate.Mul(math.NewIntFromUint64(uint64(authz.MaxDeviationBps)))
		if lhs.GT(rhs) {
			return nil, errorsmod.Wrapf(types.ErrRateDeviationExceeded, "rate change exceeds max_deviation_bps %d", authz.MaxDeviationBps)
		}
	}

	// [MOD-XR-MSG-2-3] update rate; recompute expires from the stored validity_duration.
	xr.Rate = msg.Rate
	xr.Expires = now.Add(xr.ValidityDuration)
	xr.Updated = now

	// Save updated exchange rate
	if err := ms.ExchangeRates.Set(ctx, msg.Id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store updated exchange rate")
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateExchangeRate,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyRate, msg.Rate),
		),
	)

	return &types.MsgUpdateExchangeRateResponse{}, nil
}
