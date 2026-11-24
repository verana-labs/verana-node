package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// [MOD-GF-MSG-5] Update Module Parameters
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "expected %s got %s", ms.GetAuthority(), msg.Authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := ms.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	ev := events.New(types.ModuleName, types.EventTypeUpdateModuleParameters)
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(ctx)
	return &types.MsgUpdateParamsResponse{}, nil
}
