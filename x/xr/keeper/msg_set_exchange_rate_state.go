package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// SetExchangeRateState implements [MOD-XR-MSG-3] Set Exchange Rate State.
func (ms msgServer) SetExchangeRateState(ctx context.Context, msg *types.MsgSetExchangeRateState) (*types.MsgSetExchangeRateStateResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := ms.checkAuthority(msg.Authority); err != nil {
		return nil, err
	}

	xr, err := ms.ExchangeRates.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.Id)
	}

	xr.State = msg.State
	if err := ms.ExchangeRates.Set(ctx, msg.Id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate")
	}

	ev := events.New(types.ModuleName, types.EventTypeSetExchangeRateState).Uint(types.AttributeKeyXrID, msg.Id)
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(sdk.UnwrapSDKContext(ctx))

	return &types.MsgSetExchangeRateStateResponse{}, nil
}
