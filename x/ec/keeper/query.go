package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/ec/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct{ k Keeper }

func NewQueryServerImpl(k Keeper) types.QueryServer { return queryServer{k: k} }

// GetEcosystem implements MOD-ES-QRY-1.
func (qs queryServer) GetEcosystem(ctx context.Context, req *types.QueryGetEcosystemRequest) (*types.QueryGetEcosystemResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	ec, err := qs.k.Ecosystem.Get(ctx, req.Id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "ecosystem not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	out, err := qs.buildWithVersions(ctx, ec, req.ActiveGfOnly, req.PreferredLanguage)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetEcosystemResponse{Ecosystem: *out}, nil
}

// ListEcosystems implements MOD-ES-QRY-2. Default ordering when
// modified_after is unset is by `id` ASC (cheap, stable, deterministic);
// when modified_after is set, results are sorted by `modified` DESC per
// spec MSG-2-3.
func (qs queryServer) ListEcosystems(ctx context.Context, req *types.QueryListEcosystemsRequest) (*types.QueryListEcosystemsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1024")
	}

	// Filter cheaply first, then sort and truncate, so the expensive nested
	// GFV/GFD hydration runs only for the page actually returned.
	var ecs []types.Ecosystem
	err := qs.k.Ecosystem.Walk(ctx, nil, func(_ uint64, ec types.Ecosystem) (bool, error) {
		if req.CorporationId != 0 && ec.CorporationId != req.CorporationId {
			return false, nil
		}
		if req.ModifiedAfter != nil && !ec.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}
		ecs = append(ecs, ec)
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.ModifiedAfter != nil {
		sort.Slice(ecs, func(i, j int) bool { return ecs[i].Modified.After(ecs[j].Modified) })
	} else {
		sort.Slice(ecs, func(i, j int) bool { return ecs[i].Id < ecs[j].Id })
	}
	if len(ecs) > int(req.ResponseMaxSize) {
		ecs = ecs[:int(req.ResponseMaxSize)]
	}

	out := make([]types.EcosystemWithVersions, 0, len(ecs))
	for _, ec := range ecs {
		ew, err := qs.buildWithVersions(ctx, ec, req.ActiveGfOnly, req.PreferredLanguage)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		out = append(out, *ew)
	}

	return &types.QueryListEcosystemsResponse{Ecosystems: out}, nil
}

// Params implements MOD-ES-QRY-3.
func (qs queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := qs.k.Params.Get(ctx)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return &types.QueryParamsResponse{Params: types.Params{}}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryParamsResponse{Params: params}, nil
}

// buildWithVersions assembles the EcosystemWithVersions response shape.
// Nested GFV+GFD come from x/gf via gfKeeper.ListVersionsByEcosystem.
func (qs queryServer) buildWithVersions(ctx context.Context, ec types.Ecosystem, activeOnly bool, preferredLang string) (*types.EcosystemWithVersions, error) {
	versions, err := qs.k.gfKeeper.ListVersionsByEcosystem(ctx, ec.Id, ec.ActiveVersion, activeOnly, preferredLang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.EcosystemWithVersions{
		Id:            ec.Id,
		Did:           ec.Did,
		CorporationId: ec.CorporationId,
		Created:       ec.Created,
		Modified:      ec.Modified,
		Archived:      ec.Archived,
		Language:      ec.Language,
		ActiveVersion: ec.ActiveVersion,
		Versions:      versions,
	}, nil
}
