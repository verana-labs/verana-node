package keeper

import (
	"context"
	"sort"
	"strings"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/gf/types"
)

func (q querier) GetGovernanceFrameworkVersion(goCtx context.Context, req *types.QueryGetGovernanceFrameworkVersionRequest) (*types.QueryGetGovernanceFrameworkVersionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	gfv, err := q.GFVersion.Get(goCtx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gfv %d not found", req.Id)
	}
	docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
	}
	return &types.QueryGetGovernanceFrameworkVersionResponse{
		Version: types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		},
	}, nil
}

func (q querier) ListGovernanceFrameworkVersions(goCtx context.Context, req *types.QueryListGovernanceFrameworkVersionsRequest) (*types.QueryListGovernanceFrameworkVersionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	hasEco := req.EcosystemId > 0
	hasCorp := req.CorporationId > 0
	if hasEco == hasCorp {
		return nil, status.Error(codes.InvalidArgument, "exactly one of ecosystem_id or corporation_id must be set")
	}
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be <= 1024")
	}

	// Spec MOD-GF-QRY-2-1: active_only returns "only the entry corresponding to
	// the subject's active_version" — resolve subject's active_version once.
	var subjectActiveVersion uint32
	if req.ActiveOnly {
		if hasEco {
			eco, ok := q.ecosystemKeeper().GetEcosystemView(goCtx, req.EcosystemId)
			if !ok {
				return nil, status.Errorf(codes.NotFound, "ecosystem %d not found", req.EcosystemId)
			}
			subjectActiveVersion = eco.ActiveVersion
		} else {
			coView, ok := q.corporationKeeper().GetByID(goCtx, req.CorporationId)
			if !ok {
				return nil, status.Errorf(codes.NotFound, "corporation %d not found", req.CorporationId)
			}
			subjectActiveVersion = coView.ActiveVersion
		}
	}

	var gfvIDs []uint64
	if hasEco {
		iter, err := q.GFVersionByEcosystem.Iterate(goCtx, collections.NewPrefixedPairRange[uint64, uint32](req.EcosystemId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "iterate: %v", err)
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			id, err := iter.Value()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "iter value: %v", err)
			}
			gfvIDs = append(gfvIDs, id)
		}
	} else {
		iter, err := q.GFVersionByCorporation.Iterate(goCtx, collections.NewPrefixedPairRange[uint64, uint32](req.CorporationId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "iterate: %v", err)
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			id, err := iter.Value()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "iter value: %v", err)
			}
			gfvIDs = append(gfvIDs, id)
		}
	}

	gfvs := make([]types.GovernanceFrameworkVersion, 0, len(gfvIDs))
	for _, id := range gfvIDs {
		gfv, err := q.GFVersion.Get(goCtx, id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "fetch gfv %d: %v", id, err)
		}
		if req.ActiveOnly && gfv.Version != subjectActiveVersion {
			continue
		}
		gfvs = append(gfvs, gfv)
	}
	// Spec MOD-GF-QRY-2-3: order by ascending version, then page.
	sort.Slice(gfvs, func(i, j int) bool { return gfvs[i].Version < gfvs[j].Version })
	if uint32(len(gfvs)) > req.ResponseMaxSize {
		gfvs = gfvs[:req.ResponseMaxSize]
	}
	// Collect documents only for the page we return.
	versions := make([]types.GovernanceFrameworkVersionWithDocs, 0, len(gfvs))
	for _, gfv := range gfvs {
		docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
		}
		versions = append(versions, types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		})
	}
	return &types.QueryListGovernanceFrameworkVersionsResponse{Versions: versions}, nil
}

func (q querier) collectDocs(ctx context.Context, gfvID uint64, preferredLang string) ([]types.GovernanceFrameworkDocument, error) {
	var out []types.GovernanceFrameworkDocument
	var preferred, lowest *types.GovernanceFrameworkDocument
	// Iterate only this version's documents via the (gfv_id, *) index.
	rng := collections.NewPrefixedPairRange[uint64, string](gfvID)
	if err := q.GFDocumentByGFVLang.Walk(ctx, rng, func(_ collections.Pair[uint64, string], id uint64) (bool, error) {
		d, err := q.GFDocument.Get(ctx, id)
		if err != nil {
			return true, err
		}
		if preferredLang != "" {
			if preferred == nil && strings.EqualFold(d.Language, preferredLang) {
				cp := d
				preferred = &cp
			}
			if lowest == nil || d.Id < lowest.Id {
				cp := d
				lowest = &cp
			}
			return false, nil
		}
		out = append(out, d)
		return false, nil
	}); err != nil {
		return nil, err
	}
	// Spec MOD-GF-QRY: exactly one document per version when a preferred language
	// is set — the match if present, else the lowest-id document.
	if preferredLang != "" {
		switch {
		case preferred != nil:
			return []types.GovernanceFrameworkDocument{*preferred}, nil
		case lowest != nil:
			return []types.GovernanceFrameworkDocument{*lowest}, nil
		default:
			return nil, nil
		}
	}
	return out, nil
}
