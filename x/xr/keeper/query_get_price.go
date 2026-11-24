package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func (q queryServer) GetPrice(ctx context.Context, req *types.QueryGetPriceRequest) (*types.QueryGetPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	amount, ok := math.NewIntFromString(req.Amount)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid amount: must be an integer")
	}

	price, err := q.k.GetPrice(ctx, req.BaseAssetType, req.BaseAsset, req.QuoteAssetType, req.QuoteAsset, amount)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetPriceResponse{Price: price.String()}, nil
}

// GetPrice converts amount base units of the base asset into the quote asset [MOD-XR-QRY-3-3].
func (k Keeper) GetPrice(ctx context.Context, baseAssetType cstypes.PricingAssetType, baseAsset string, quoteAssetType cstypes.PricingAssetType, quoteAsset string, amount math.Int) (math.Int, error) {
	if amount.IsNegative() {
		return math.Int{}, errorsmod.Wrapf(types.ErrInvalidAmount, "amount must not be negative, got %s", amount)
	}
	if err := types.ValidateAsset(baseAssetType, baseAsset); err != nil {
		return math.Int{}, errorsmod.Wrap(err, "base")
	}
	if err := types.ValidateAsset(quoteAssetType, quoteAsset); err != nil {
		return math.Int{}, errorsmod.Wrap(err, "quote")
	}
	if amount.IsZero() || (baseAssetType == quoteAssetType && baseAsset == quoteAsset) {
		return amount, nil
	}

	_, xr, err := k.findExchangeRate(ctx, baseAssetType, baseAsset, quoteAssetType, quoteAsset)
	if err != nil {
		return math.Int{}, err
	}
	if err := checkUsable(ctx, &xr); err != nil {
		return math.Int{}, err
	}

	product, err := amount.SafeMul(math.NewIntFromUint64(xr.Rate))
	if err != nil {
		return math.Int{}, errorsmod.Wrap(types.ErrInvalidAmount, err.Error())
	}
	divisor := uint64(1)
	for range xr.RateScale {
		divisor *= 10
	}
	return product.Quo(math.NewIntFromUint64(divisor)), nil
}
