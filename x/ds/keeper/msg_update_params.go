package keeper

import (
	"bytes"
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/ds/types"
)

// [MOD-DS-MSG-1] Update Module Parameters
func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	authority, err := k.addressCodec.StringToBytes(req.Signer)
	if err != nil || !bytes.Equal(k.GetAuthority(), authority) {
		expected, _ := k.addressCodec.BytesToString(k.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", expected, req.Signer)
	}
	if err := k.validateParams(req.Params); err != nil {
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

// validateParams adds to Params.Validate that the foundation account can receive its payout.
func (k Keeper) validateParams(p types.Params) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if k.bank.BlockedAddr(sdk.MustAccAddressFromBech32(p.FoundationAccount)) {
		return errorsmod.Wrapf(types.ErrInvalidParams, "foundation_account %s cannot receive funds", p.FoundationAccount)
	}
	return nil
}
