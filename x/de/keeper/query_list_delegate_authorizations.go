package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/de/types"
)

func (q queryServer) ListDelegateAuthorizations(ctx context.Context, req *types.QueryListDelegateAuthorizationsRequest) (*types.QueryListDelegateAuthorizationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// [MOD-DE-QRY-1-2] Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	// [MOD-DE-QRY-1-3] Walk through all delegate authorizations and apply filters
	var results []types.DelegateAuthorization

	err := q.k.DelegateAuthorizations.Walk(ctx, nil, func(_ uint64, da types.DelegateAuthorization) (bool, error) {
		// Filter by corporation_id if specified
		if req.CorporationId != 0 && da.CorporationId != req.CorporationId {
			return false, nil
		}
		// Filter by delegate if specified
		if req.Delegate != "" && da.Delegate != req.Delegate {
			return false, nil
		}

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
