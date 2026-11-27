package keeper

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/xr/types"
)

// GrantExchangeRateAuthorization implements [MOD-XR-MSG-4] Grant Exchange Rate Authorization.
func (ms msgServer) GrantExchangeRateAuthorization(ctx context.Context, msg *types.MsgGrantExchangeRateAuthorization) (*types.MsgGrantExchangeRateAuthorizationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// [MOD-XR-MSG-4-2-1] authority MUST equal the council account.
	authority, err := ms.addressCodec.StringToBytes(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}
	if !bytes.Equal(ms.GetAuthority(), authority) {
		expected, _ := ms.addressCodec.BytesToString(ms.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", expected, msg.Signer)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// xr_id MUST refer to an existing ExchangeRate.
	if _, err := ms.ExchangeRates.Get(ctx, msg.XrId); err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.XrId)
	}

	// expiration MUST be in the future.
	if !msg.Expiration.After(now) {
		return nil, types.ErrInvalidExpiration
	}

	// Build the authorization record (zero min_interval / max_deviation_bps = unset).
	authz := types.ExchangeRateAuthorization{
		XrId:            msg.XrId,
		Feeder:          msg.Feeder,
		Expiration:      *msg.Expiration,
		MaxDeviationBps: msg.MaxDeviationBps,
	}
	if msg.MinInterval != nil {
		authz.MinInterval = *msg.MinInterval
	}

	key := collections.Join(msg.XrId, msg.Feeder)
	if err := ms.ExchangeRateAuthorizations.Set(ctx, key, authz); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate authorization")
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantExchangeRateAuthz,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", msg.XrId)),
			sdk.NewAttribute(types.AttributeKeyFeeder, msg.Feeder),
			sdk.NewAttribute(types.AttributeKeyExpiration, authz.Expiration.UTC().Format(time.RFC3339Nano)),
			sdk.NewAttribute(types.AttributeKeyMinInterval, fmt.Sprintf("%d", authz.MinInterval)),
			sdk.NewAttribute(types.AttributeKeyMaxDeviationBps, fmt.Sprintf("%d", authz.MaxDeviationBps)),
		),
	)

	return &types.MsgGrantExchangeRateAuthorizationResponse{}, nil
}
