package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/pp/types"
)

// resolvePricing maps a schema's pricing asset to the on-chain settlement of a
// fee amount, per the [MOD-PP] fee model. It returns the fee expressed in its
// settlement denom, that denom, and the fee converted to the native denom (the
// basis for trust-deposit and agent-reward math, which are always native).
//
//   - (COIN, native): settled native, no conversion.
//   - (TU, "tu"):      converted to native via getPrice, settled native.
//   - (COIN, other):   settled in that denom; only the native basis is converted.
//   - (FIAT, *):       settled off-chain (0 on-chain); native basis converted.
func (ms msgServer) resolvePricing(ctx sdk.Context, cs cstypes.CredentialSchema, amount uint64) (feeInDenom uint64, feeDenom string, nativeBasis uint64, err error) {
	native := types.BondDenom
	switch cs.PricingAssetType {
	case cstypes.PricingAssetType_COIN:
		if cs.PricingAsset == native {
			return amount, native, amount, nil
		}
		basis, err := ms.toNative(ctx, cs.PricingAssetType, cs.PricingAsset, amount)
		if err != nil {
			return 0, "", 0, err
		}
		return amount, cs.PricingAsset, basis, nil
	case cstypes.PricingAssetType_TU:
		conv, err := ms.toNative(ctx, cs.PricingAssetType, cs.PricingAsset, amount)
		if err != nil {
			return 0, "", 0, err
		}
		return conv, native, conv, nil
	case cstypes.PricingAssetType_FIAT:
		basis, err := ms.toNative(ctx, cs.PricingAssetType, cs.PricingAsset, amount)
		if err != nil {
			return 0, "", 0, err
		}
		return 0, native, basis, nil
	default:
		return 0, "", 0, fmt.Errorf("unsupported pricing_asset_type %d", cs.PricingAssetType)
	}
}

// feeDenomForSchema returns the denom a schema's fees settle in on-chain: the
// pricing asset itself for an arbitrary COIN, otherwise the native denom (native
// COIN settles native; TU and FIAT convert to native). Mirrors the feeDenom that
// resolvePricing returns, without needing an amount to convert.
func feeDenomForSchema(cs cstypes.CredentialSchema) string {
	if cs.PricingAssetType == cstypes.PricingAssetType_COIN && cs.PricingAsset != types.BondDenom {
		return cs.PricingAsset
	}
	return types.BondDenom
}

// toNative converts amount of (assetType, asset) into the native denom via x/xr.
func (ms msgServer) toNative(ctx sdk.Context, assetType cstypes.PricingAssetType, asset string, amount uint64) (uint64, error) {
	if amount == 0 {
		return 0, nil
	}
	out, err := ms.exchangeRateKeeper.GetPrice(ctx, assetType, asset, cstypes.PricingAssetType_COIN, types.BondDenom, math.NewIntFromUint64(amount).String())
	if err != nil {
		return 0, fmt.Errorf("price conversion to native denom failed: %w", err)
	}
	v, ok := math.NewIntFromString(out)
	if !ok || !v.IsUint64() {
		return 0, fmt.Errorf("invalid converted price %q", out)
	}
	return v.Uint64(), nil
}
