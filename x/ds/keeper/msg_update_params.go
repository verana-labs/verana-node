package keeper

import (
	"bytes"
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/ds/types"
)

// [MOD-DS-MSG-1] Update Module Parameters
func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	authority, err := k.addressCodec.StringToBytes(req.Signer)
	if err != nil || !bytes.Equal(k.GetAuthority(), authority) {
		expected, _ := k.addressCodec.BytesToString(k.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", expected, req.Signer)
	}
	if err := req.Params.Validate(); err != nil {
		return nil, err
	}
	if err := k.Params.Set(ctx, req.Params); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateParams,
		append([]sdk.Attribute{sdk.NewAttribute(types.AttributeKeySigner, req.Signer)}, req.Params.EventAttributes()...)...,
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
