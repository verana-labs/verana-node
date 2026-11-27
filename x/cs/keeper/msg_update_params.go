package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/cs/types"
)

// [MOD-CS-MSG-4] Update Module Parameters
func (ms msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.GetAuthority() != req.Signer {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", ms.GetAuthority(), req.Signer)
	}
	if err := req.Params.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := ms.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateParams,
		append([]sdk.Attribute{sdk.NewAttribute(types.AttributeKeySigner, req.Signer)}, req.Params.EventAttributes()...)...,
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
