package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/xr/types"
)

// RevokeExchangeRateAuthorization implements [MOD-XR-MSG-5] Revoke Exchange Rate Authorization.
func (ms msgServer) RevokeExchangeRateAuthorization(ctx context.Context, msg *types.MsgRevokeExchangeRateAuthorization) (*types.MsgRevokeExchangeRateAuthorizationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// authority MUST equal the council account.
	authority, err := ms.addressCodec.StringToBytes(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}
	if !bytes.Equal(ms.GetAuthority(), authority) {
		expected, _ := ms.addressCodec.BytesToString(ms.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", expected, msg.Signer)
	}

	// (xr_id, feeder) MUST exist.
	key := collections.Join(msg.XrId, msg.Feeder)
	if _, err := ms.ExchangeRateAuthorizations.Get(ctx, key); err != nil {
		return nil, errorsmod.Wrapf(types.ErrAuthorizationNotFound, "no authorization for xr_id %d feeder %s", msg.XrId, msg.Feeder)
	}

	if err := ms.ExchangeRateAuthorizations.Remove(ctx, key); err != nil {
		return nil, errorsmod.Wrap(err, "failed to delete exchange rate authorization")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeExchangeRateAuthz,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", msg.XrId)),
			sdk.NewAttribute(types.AttributeKeyFeeder, msg.Feeder),
		),
	)

	return &types.MsgRevokeExchangeRateAuthorizationResponse{}, nil
}
