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

// GetSponsorship implements [MOD-DE-QRY-5].
func (q queryServer) GetSponsorship(ctx context.Context, req *types.QueryGetSponsorshipRequest) (*types.QueryGetSponsorshipResponse, error) {
	if req == nil || req.SponsorCorporationId == 0 || req.SponsoredCorporationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "sponsor_corporation_id and sponsored_corporation_id are required")
	}
	s, err := q.k.Sponsorships.Get(ctx, collections.Join(req.SponsorCorporationId, req.SponsoredCorporationId))
	if errors.Is(err, collections.ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, "sponsorship (%d, %d) not found", req.SponsorCorporationId, req.SponsoredCorporationId)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryGetSponsorshipResponse{Sponsorship: s}, nil
}

// ListSponsorships implements [MOD-DE-QRY-6].
func (q queryServer) ListSponsorships(ctx context.Context, req *types.QueryListSponsorshipsRequest) (*types.QueryListSponsorshipsResponse, error) {
	if req == nil || req.SponsorCorporationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "sponsor_corporation_id is required")
	}
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	var out []types.ListedSponsorship
	rng := collections.NewPrefixedPairRange[uint64, uint64](req.SponsorCorporationId)
	err := q.k.Sponsorships.Walk(ctx, rng, func(_ collections.Pair[uint64, uint64], s types.Sponsorship) (bool, error) {
		sponsored, err := q.k.corporationKeeper().ResolveCorporationByID(ctx, s.SponsoredCorporationId)
		if err != nil {
			return true, err
		}
		active := sponsored.SponsorCorporationId == s.SponsorCorporationId && sponsorshipLive(&s, now)
		out = append(out, types.ListedSponsorship{Sponsorship: s, Active: active})
		return len(out) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryListSponsorshipsResponse{Sponsorships: out}, nil
}
