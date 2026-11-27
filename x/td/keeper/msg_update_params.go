package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/td/types"
)

// [MOD-TD-MSG-4] Update Module Parameters
func (ms msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.GetAuthority(), req.Authority)
	}
	if err := req.Params.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	current := ms.GetParams(ctx)
	if err := req.Params.CheckImmutable(current); err != nil {
		return nil, err
	}

	// [MOD-TD-MSG-4-3] a new tu_decay_rate waits for the next epoch boundary;
	// the rate in force is not a new value and keeps a pending one.
	next := req.Params
	next.TuDecayRate = current.TuDecayRate
	if !req.Params.TuDecayRate.Equal(current.TuDecayRate) {
		if err := ms.PendingTuDecayRate.Set(ctx, req.Params.TuDecayRate); err != nil {
			return nil, err
		}
	}
	if err := ms.SetParams(ctx, next); err != nil {
		return nil, err
	}

	pending := ""
	if rate, found, err := ms.GetPendingTuDecayRate(ctx); err != nil {
		return nil, err
	} else if found {
		pending = rate.String()
	}
	attrs := append([]sdk.Attribute{sdk.NewAttribute(types.AttributeKeyAuthority, req.Authority)}, next.EventAttributes()...)
	attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyPendingTuDecayRate, pending))
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeUpdateParams, attrs...))

	return &types.MsgUpdateParamsResponse{}, nil
}
