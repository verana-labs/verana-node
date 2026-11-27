package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// RevokeExchangeRateAuthorization implements [MOD-XR-MSG-5] Revoke Exchange Rate Authorization.
func (ms msgServer) RevokeExchangeRateAuthorization(ctx context.Context, msg *types.MsgRevokeExchangeRateAuthorization) (*types.MsgRevokeExchangeRateAuthorizationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := ms.checkAuthority(msg.Signer); err != nil {
		return nil, err
	}
	feederBz, err := ms.addressCodec.StringToBytes(msg.Feeder)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid feeder address")
	}
	feeder := sdk.AccAddress(feederBz)

	key := collections.Join(msg.XrId, feeder)
	has, err := ms.ExchangeRateAuthorizations.Has(ctx, key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errorsmod.Wrapf(types.ErrAuthorizationNotFound, "no authorization for xr_id %d feeder %s", msg.XrId, msg.Feeder)
	}
	if err := ms.ExchangeRateAuthorizations.Remove(ctx, key); err != nil {
		return nil, errorsmod.Wrap(err, "failed to delete exchange rate authorization")
	}

	ev := events.New(types.ModuleName, types.EventTypeRevokeExchangeRateAuthorization).
		Uint(types.AttributeKeyXrID, msg.XrId).
		Addr(types.AttributeKeyFeeder, feeder)
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(sdk.UnwrapSDKContext(ctx))

	return &types.MsgRevokeExchangeRateAuthorizationResponse{}, nil
}
