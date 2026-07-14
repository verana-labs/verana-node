package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/verana-labs/verana-node/x/co/types"
)

func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if ms.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner,
			"invalid authority; expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := ms.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateParams,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
	))
	return &types.MsgUpdateParamsResponse{}, nil
}
