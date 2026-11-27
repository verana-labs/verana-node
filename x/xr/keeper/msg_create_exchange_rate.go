package keeper

import (
	"context"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// CreateExchangeRate implements [MOD-XR-MSG-1] Create Exchange Rate.
func (ms msgServer) CreateExchangeRate(ctx context.Context, msg *types.MsgCreateExchangeRate) (*types.MsgCreateExchangeRateResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := ms.checkAuthority(msg.Signer); err != nil {
		return nil, err
	}

	if err := ms.assertCoinAssetExists(ctx, msg.BaseAssetType, msg.BaseAsset, "base"); err != nil {
		return nil, err
	}
	if err := ms.assertCoinAssetExists(ctx, msg.QuoteAssetType, msg.QuoteAsset, "quote"); err != nil {
		return nil, err
	}

	_, _, err := ms.findExchangeRate(ctx, msg.BaseAssetType, msg.BaseAsset, msg.QuoteAssetType, msg.QuoteAsset)
	if err == nil {
		return nil, types.ErrDuplicatePair
	}
	if !isNotFound(err) {
		return nil, err
	}

	id, err := ms.GetNextID(ctx, types.CounterKeyExchangeRate)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to generate exchange rate ID")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime().Unix()
	validity := uint32(msg.ValidityDuration / time.Second)
	xr := types.ExchangeRateRecord{
		BaseAssetType:    msg.BaseAssetType,
		BaseAsset:        msg.BaseAsset,
		QuoteAssetType:   msg.QuoteAssetType,
		QuoteAsset:       msg.QuoteAsset,
		Rate:             msg.Rate,
		RateScale:        msg.RateScale,
		ValidityDuration: validity,
		Updated:          now,
		Expires:          now + int64(validity),
		State:            false,
	}
	if err := ms.ExchangeRates.Set(ctx, id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate")
	}

	ev := events.New(types.ModuleName, types.EventTypeCreateExchangeRate).
		Uint(types.AttributeKeyXrID, id).
		Time(types.AttributeKeyUpdated, time.Unix(xr.Updated, 0)).
		Time(types.AttributeKeyExpires, time.Unix(xr.Expires, 0))
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(sdkCtx)

	return &types.MsgCreateExchangeRateResponse{Id: id}, nil
}

// assertCoinAssetExists implements the [MOD-XR-MSG-1-2-1] on-chain existence check
// for a COIN asset: the denom must be recognized by the chain (have supply).
// factory/ denoms are rejected — this chain has no tokenfactory module.
func (ms msgServer) assertCoinAssetExists(ctx context.Context, at cstypes.PricingAssetType, asset, side string) error {
	if at != cstypes.PricingAssetType_COIN {
		return nil
	}
	if strings.HasPrefix(asset, "factory/") {
		return errorsmod.Wrapf(types.ErrInvalidRequest, "%s_asset %q: tokenfactory denoms are not supported", side, asset)
	}
	if !ms.bankKeeper.HasSupply(ctx, asset) {
		return errorsmod.Wrapf(types.ErrInvalidRequest, "%s_asset %q does not exist on-chain", side, asset)
	}
	return nil
}
