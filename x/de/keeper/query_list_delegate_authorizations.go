package keeper

import (
	"context"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/de/types"
)

// ListDelegateAuthorizations implements [MOD-DE-QRY-1]: the authorizations of a
// corporation, by key prefix.
func (q queryServer) ListDelegateAuthorizations(ctx context.Context, req *types.QueryListDelegateAuthorizationsRequest) (*types.QueryListDelegateAuthorizationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.CorporationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "corporation_id is mandatory")
	}

	// [MOD-DE-QRY-1-2] Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	// [MOD-DE-QRY-1-3]
	var results []types.DelegateAuthorization
	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](req.CorporationId)
	err := q.k.DelegateAuthorizations.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], da types.DelegateAuthorization) (bool, error) {
		withKey(&da, key.K1(), key.K2())
		results = append(results, da)
		return len(results) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListDelegateAuthorizationsResponse{
		DelegateAuthorizations: results,
	}, nil
}
