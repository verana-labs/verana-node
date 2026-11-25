package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// [MOD-XR-MSG-6] Update Module Parameters
func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := k.checkAuthority(req.Authority); err != nil {
		return nil, err
	}
	if err := req.Params.Validate(); err != nil {
		return nil, err
	}
	if err := k.Params.Set(ctx, req.Params); err != nil {
		return nil, err
	}

	ev := events.New(types.ModuleName, types.EventTypeUpdateModuleParameters)
	if err := ev.Msg(k.cdc, req); err != nil {
		return nil, err
	}
	ev.Emit(sdk.UnwrapSDKContext(ctx))

	return &types.MsgUpdateParamsResponse{}, nil
}
