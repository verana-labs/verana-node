package keeper

import (
	"context"
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
	docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage, q.subjectLanguage(goCtx, gfv))
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

// GetEcosystemAcceptance implements MOD-GF-QRY-2.
func (q querier) GetEcosystemAcceptance(goCtx context.Context, req *types.QueryGetEcosystemAcceptanceRequest) (*types.QueryGetEcosystemAcceptanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	acc, found, err := q.getEcosystemAcceptance(goCtx, req.EcosystemId, req.CorporationId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get acceptance: %v", err)
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "no acceptance of ecosystem %d by corporation %d", req.EcosystemId, req.CorporationId)
	}
	return &types.QueryGetEcosystemAcceptanceResponse{Acceptance: acc}, nil
}

// subjectLanguage returns the language of the owner of gfv, empty for a
// network version.
func (k Keeper) subjectLanguage(ctx context.Context, gfv types.GovernanceFrameworkVersion) string {
	switch {
	case gfv.EcosystemId > 0:
		e, _ := k.ecosystemKeeper().GetEcosystemView(ctx, gfv.EcosystemId)
		return e.Language
	case gfv.CorporationId > 0:
		co, _ := k.corporationKeeper().GetByID(ctx, gfv.CorporationId)
		return co.Language
	}
	return ""
}

// collectDocs returns the documents of a version. With preferredLang set it
// returns one: the one in preferredLang, else in subjectLang, else the earliest.
func (k Keeper) collectDocs(ctx context.Context, gfvID uint64, preferredLang, subjectLang string) ([]types.GovernanceFrameworkDocument, error) {
	var out []types.GovernanceFrameworkDocument
	var preferred, subject, earliest *types.GovernanceFrameworkDocument
	rng := collections.NewPrefixedPairRange[uint64, string](gfvID)
	if err := k.GFDocumentByGFVLang.Walk(ctx, rng, func(_ collections.Pair[uint64, string], id uint64) (bool, error) {
		d, err := k.GFDocument.Get(ctx, id)
		if err != nil {
			return true, err
		}
		if preferredLang == "" {
			out = append(out, d)
			return false, nil
		}
		switch {
		case preferred == nil && strings.EqualFold(d.Language, preferredLang):
			preferred = &d
		case subject == nil && subjectLang != "" && strings.EqualFold(d.Language, subjectLang):
			subject = &d
		}
		if earliest == nil || d.Created.Before(earliest.Created) || (d.Created.Equal(earliest.Created) && d.Id < earliest.Id) {
			earliest = &d
		}
		return false, nil
	}); err != nil {
		return nil, err
	}
	if preferredLang == "" {
		return out, nil
	}
	for _, d := range []*types.GovernanceFrameworkDocument{preferred, subject, earliest} {
		if d != nil {
			return []types.GovernanceFrameworkDocument{*d}, nil
		}
	}
	return nil, nil
}
