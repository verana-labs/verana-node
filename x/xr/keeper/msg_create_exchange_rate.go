package keeper

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func (ms msgServer) CreateExchangeRate(ctx context.Context, msg *types.MsgCreateExchangeRate) (*types.MsgCreateExchangeRateResponse, error) {
	// Validate basic fields
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Load and check params
	params, err := ms.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load params")
	}
	if msg.ValidityDuration > params.MaxValidityDuration {
		return nil, errorsmod.Wrapf(types.ErrInvalidRequest, "validity_duration %s exceeds max_validity_duration %s", msg.ValidityDuration, params.MaxValidityDuration)
	}

	// Governance authority check
	authority, err := ms.addressCodec.StringToBytes(msg.Authority)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	if !bytes.Equal(ms.GetAuthority(), authority) {
		expectedAuthorityStr, _ := ms.addressCodec.BytesToString(ms.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", expectedAuthorityStr, msg.Authority)
	}

	// [MOD-XR-MSG-1-2-1] COIN assets MUST exist on-chain.
	if err := ms.assertCoinAssetExists(ctx, msg.BaseAssetType, msg.BaseAsset, "base"); err != nil {
		return nil, err
	}
	if err := ms.assertCoinAssetExists(ctx, msg.QuoteAssetType, msg.QuoteAsset, "quote"); err != nil {
		return nil, err
	}

	// Check pair uniqueness
	pairKey := buildPairKey(msg.BaseAssetType, msg.BaseAsset, msg.QuoteAssetType, msg.QuoteAsset)
	_, err = ms.PairIndex.Get(ctx, pairKey)
	if err == nil {
		return nil, errorsmod.Wrapf(types.ErrDuplicatePair, "exchange rate pair already exists for %s", pairKey)
	}

	// Auto-generate ID
	id, err := ms.GetNextID(ctx, types.CounterKeyExchangeRate)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to generate exchange rate ID")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()

	// Create ExchangeRate entry
	xr := types.ExchangeRate{
		Id:               id,
		BaseAssetType:    msg.BaseAssetType,
		BaseAsset:        msg.BaseAsset,
		QuoteAssetType:   msg.QuoteAssetType,
		QuoteAsset:       msg.QuoteAsset,
		Rate:             msg.Rate,
		RateScale:        msg.RateScale,
		ValidityDuration: msg.ValidityDuration,
		Expires:          blockTime.Add(msg.ValidityDuration),
		// [MOD-XR-MSG-1-3] entry starts disabled; enabled via gov SetExchangeRateState.
		State:   false,
		Updated: blockTime,
	}

	// Store in collection
	if err := ms.ExchangeRates.Set(ctx, id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate")
	}

	// Store pair index for uniqueness
	if err := ms.PairIndex.Set(ctx, pairKey, id); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store pair index")
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateExchangeRate,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyBaseAssetType, msg.BaseAssetType.String()),
			sdk.NewAttribute(types.AttributeKeyBaseAsset, msg.BaseAsset),
			sdk.NewAttribute(types.AttributeKeyQuoteAssetType, msg.QuoteAssetType.String()),
			sdk.NewAttribute(types.AttributeKeyQuoteAsset, msg.QuoteAsset),
			sdk.NewAttribute(types.AttributeKeyRate, msg.Rate),
			sdk.NewAttribute(types.AttributeKeyRateScale, fmt.Sprintf("%d", msg.RateScale)),
			sdk.NewAttribute(types.AttributeKeyValidityDuration, msg.ValidityDuration.String()),
			sdk.NewAttribute(types.AttributeKeyExpires, xr.Expires.UTC().Format(time.RFC3339Nano)),
		),
	)

	return &types.MsgCreateExchangeRateResponse{Id: id}, nil
}

// buildPairKey uses '|' as the delimiter — illegal in SDK denoms, so distinct
// asset pairs can never collide into the same key (unlike ':', which is legal
// inside a denom).
func buildPairKey(baseType cstypes.PricingAssetType, baseAsset string, quoteType cstypes.PricingAssetType, quoteAsset string) string {
	return fmt.Sprintf("%d|%s|%d|%s", baseType, baseAsset, quoteType, quoteAsset)
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
