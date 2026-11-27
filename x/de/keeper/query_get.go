package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GetDelegateAuthorization implements [MOD-DE-QRY-3]: fetch the
// DelegateAuthorization of (corporation_id, delegate).
func (q queryServer) GetDelegateAuthorization(ctx context.Context, req *types.QueryGetDelegateAuthorizationRequest) (*types.QueryGetDelegateAuthorizationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.CorporationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "corporation_id is mandatory")
	}
	delegate, err := sdk.AccAddressFromBech32(req.Delegate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid delegate: %s", err)
	}

	da, found, err := q.k.getDelegateAuthorization(ctx, req.CorporationId, delegate)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "delegate authorization (%d, %s) not found", req.CorporationId, req.Delegate)
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
