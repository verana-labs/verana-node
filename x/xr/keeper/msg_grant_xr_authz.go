package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// GrantExchangeRateAuthorization implements [MOD-XR-MSG-4] Grant Exchange Rate Authorization.
func (ms msgServer) GrantExchangeRateAuthorization(ctx context.Context, msg *types.MsgGrantExchangeRateAuthorization) (*types.MsgGrantExchangeRateAuthorizationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := ms.checkAuthority(msg.Authority); err != nil {
		return nil, err
	}
	feederBz, err := ms.addressCodec.StringToBytes(msg.Feeder)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid feeder address")
	}
	feeder := sdk.AccAddress(feederBz)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	xr, err := ms.ExchangeRates.Get(ctx, msg.XrId)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.XrId)
	}
	if msg.Expiration.Unix() <= sdkCtx.BlockTime().Unix() {
		return nil, types.ErrInvalidExpiration
	}
	if msg.MaxDeviationBps == 0 && ms.isGenesisPair(ctx, &xr) {
		return nil, errorsmod.Wrap(types.ErrInvalidMaxDeviation, "max_deviation_bps is mandatory on the genesis exchange rate")
	}

	authz := types.ExchangeRateAuthorizationRecord{
		Expiration:      msg.Expiration.Unix(),
		MaxDeviationBps: msg.MaxDeviationBps,
	}
	if msg.MinInterval != nil {
		authz.MinInterval = uint32(*msg.MinInterval / time.Second)
	}
	if err := ms.ExchangeRateAuthorizations.Set(ctx, collections.Join(msg.XrId, feeder), authz); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate authorization")
	}

	ev := events.New(types.ModuleName, types.EventTypeGrantExchangeRateAuthorization).
		Uint(types.AttributeKeyXrID, msg.XrId).
		Addr(types.AttributeKeyFeeder, feeder)
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(sdkCtx)

	return &types.MsgGrantExchangeRateAuthorizationResponse{}, nil
}
