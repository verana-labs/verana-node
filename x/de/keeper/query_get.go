package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GetDelegateAuthorization implements [MOD-DE-QRY-3]: fetch an
// DelegateAuthorization by id.
func (q queryServer) GetDelegateAuthorization(ctx context.Context, req *types.QueryGetDelegateAuthorizationRequest) (*types.QueryGetDelegateAuthorizationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	da, err := q.k.DelegateAuthorizations.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "delegate authorization %d not found", req.Id)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetDelegateAuthorizationResponse{DelegateAuthorization: da}, nil
}

// GetVSOperatorAuthorization implements [MOD-DE-QRY-4]: fetch a
// VSOperatorAuthorization (including its records) by id.
func (q queryServer) GetVSOperatorAuthorization(ctx context.Context, req *types.QueryGetVSOperatorAuthorizationRequest) (*types.QueryGetVSOperatorAuthorizationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	vsoa, err := q.k.VSOperatorAuthorizations.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "vs operator authorization %d not found", req.Id)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetVSOperatorAuthorizationResponse{VsOperatorAuthorization: vsoa}, nil
}
