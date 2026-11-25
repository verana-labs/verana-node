package keeper

import (
	"context"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

// GetExchangeRate implements [MOD-XR-QRY-1] Get Exchange Rate.
func (q queryServer) GetExchangeRate(ctx context.Context, req *types.QueryGetExchangeRateRequest) (*types.QueryGetExchangeRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	id := req.Id
	var (
		xr  types.ExchangeRateRecord
		err error
	)
	if id > 0 {
		xr, err = q.k.ExchangeRates.Get(ctx, id)
	} else {
		if req.BaseAssetType == cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED ||
			req.BaseAsset == "" ||
			req.QuoteAssetType == cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED ||
			req.QuoteAsset == "" {
			return nil, status.Error(codes.InvalidArgument, "must provide id or all of base_asset_type, base_asset, quote_asset_type, quote_asset")
		}
		id, xr, err = q.k.findExchangeRate(ctx, req.BaseAssetType, req.BaseAsset, req.QuoteAssetType, req.QuoteAsset)
	}
	if err != nil && !isNotFound(err) {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err != nil ||
		(req.State == types.StateFilter_STATE_FILTER_ACTIVE && !xr.State) ||
		(req.State == types.StateFilter_STATE_FILTER_INACTIVE && xr.State) ||
		(req.Expires != nil && xr.Expires <= req.Expires.Unix()) {
		return nil, status.Error(codes.NotFound, "exchange rate not found")
	}

	var authorizations []types.ExchangeRateAuthorization
	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](id)
	if err := q.k.ExchangeRateAuthorizations.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], v types.ExchangeRateAuthorizationRecord) (bool, error) {
		feeder, err := q.k.addressCodec.BytesToString(key.K2())
		if err != nil {
			return true, err
		}
		authorizations = append(authorizations, v.Authorization(id, feeder))
		return false, nil
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetExchangeRateResponse{ExchangeRate: xr.ExchangeRate(id), Authorizations: authorizations}, nil
}
