package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// subjectKind is an internal tag for which owner the request targets.
type subjectKind int

const (
	subjectEcosystem subjectKind = iota
	subjectCorporation
	subjectNetwork
)

// resolvedSubject holds the validated owner of a GF operation.
type resolvedSubject struct {
	kind          subjectKind
	ecosystemID   uint64
	corporationID uint64
	language      string
	activeVersion uint32
}

// authorize runs AUTHZ-CHECK-5 (with GF-CHECK-1), then AUTHZ-CHECK-1 for the
// delegate, and returns the resolved corporation.
func (ms msgServer) authorize(ctx sdk.Context, corporation, delegate string, msgType detypes.MsgType) (types.CorporationView, error) {
	coView, err := ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, corporation)
	if err != nil {
		return types.CorporationView{}, err
	}
	if err := ms.delegationKeeper.CheckDelegateAuthorization(ctx, coView.Id, delegate, msgType, ctx.BlockTime()); err != nil {
		return types.CorporationView{}, err
	}
	return coView, nil
}

// resolveSubject implements the spec's "Define subject as ..." block for
// MOD-GF-MSG-1-2-1 and MOD-GF-MSG-2-2-1, for the corporation resolved by
// AUTHZ-CHECK-5.
func (k Keeper) resolveSubject(ctx context.Context, coView types.CorporationView, ecosystemID uint64) (resolvedSubject, error) {
	if ecosystemID != 0 {
		eco, ok := k.ecosystemKeeper().GetEcosystemView(ctx, ecosystemID)
		if !ok {
			return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "ecosystem %d", ecosystemID)
		}
		if eco.CorporationID != coView.Id {
			return resolvedSubject{}, types.ErrSubjectNotControlled
		}
		return resolvedSubject{
			kind:          subjectEcosystem,
			ecosystemID:   eco.Id,
			language:      eco.Language,
			activeVersion: eco.ActiveVersion,
		}, nil
	}

	return resolvedSubject{
		kind:          subjectCorporation,
		corporationID: coView.Id,
		language:      coView.Language,
		activeVersion: coView.ActiveVersion,
	}, nil
}

// maxVersionFor returns the highest known GFV.version for the subject, or 0 if none.
// Also returns whether a GFV with `targetVersion` already exists and its id.
func (k Keeper) maxVersionFor(ctx context.Context, sub resolvedSubject, targetVersion uint32) (maxV uint32, hasTarget bool, gfvID uint64, err error) {
	switch sub.kind {
	case subjectEcosystem:
		iter, e := k.GFVersionByEcosystem.Iterate(ctx, collections.NewPrefixedPairRange[uint64, uint32](sub.ecosystemID))
		if e != nil {
			err = e
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, e := iter.Key()
			if e != nil {
				err = e
				return
			}
			v := key.K2()
			if v > maxV {
				maxV = v
			}
			if v == targetVersion {
				hasTarget = true
				id, e := iter.Value()
				if e != nil {
					err = e
					return
				}
				gfvID = id
			}
		}
	case subjectCorporation:
		iter, e := k.GFVersionByCorporation.Iterate(ctx, collections.NewPrefixedPairRange[uint64, uint32](sub.corporationID))
		if e != nil {
			err = e
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, e := iter.Key()
			if e != nil {
				err = e
				return
			}
			v := key.K2()
			if v > maxV {
				maxV = v
			}
			if v == targetVersion {
				hasTarget = true
				id, e := iter.Value()
				if e != nil {
					err = e
					return
				}
				gfvID = id
			}
		}
	}
	return
}

// AddGovernanceFrameworkDocument implements MOD-GF-MSG-1.
func (ms msgServer) AddGovernanceFrameworkDocument(goCtx context.Context, msg *types.MsgAddGovernanceFrameworkDocument) (*types.MsgAddGovernanceFrameworkDocumentResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		gfv    types.GovernanceFrameworkVersion
		exists bool
		ng     types.NetworkGovernance
		coView types.CorporationView
		sub    resolvedSubject
		err    error
	)
	if msg.Authority != "" {
		if err := ms.checkAuthority(msg.Authority); err != nil {
			return nil, err
		}
		sub.kind = subjectNetwork
		gfv, exists, ng, err = ms.networkTarget(ctx, msg.Version)
		if err != nil {
			return nil, err
		}
	} else {
		coView, err = ms.authorize(ctx, msg.Corporation, msg.Delegate, detypes.MsgType_MSG_TYPE_ADD_GOVERNANCE_FRAMEWORK_DOCUMENT)
		if err != nil {
			return nil, err
		}
		sub, err = ms.resolveSubject(ctx, coView, msg.EcosystemId)
		if err != nil {
			return nil, err
		}
		maxV, hasTarget, existingGfvID, err := ms.maxVersionFor(ctx, sub, msg.Version)
		if err != nil {
			return nil, err
		}
		if !hasTarget && msg.Version != maxV+1 {
			return nil, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d", maxV+1)
		}
		if msg.Version <= sub.activeVersion {
			return nil, cerrors.Wrapf(types.ErrInvalidVersion, "must be greater than active_version %d", sub.activeVersion)
		}
		if hasTarget {
			gfv, err = ms.GFVersion.Get(ctx, existingGfvID)
			if err != nil {
				return nil, fmt.Errorf("fetch gfv %d: %w", existingGfvID, err)
			}
			exists = true
		}
	}

	if !exists {
		nextID, err := ms.GetNextID(ctx, "gfv")
		if err != nil {
			return nil, err
		}
		gfv = types.GovernanceFrameworkVersion{
			Id:      nextID,
			Created: ctx.BlockTime(),
			Version: msg.Version,
		}
		switch sub.kind {
		case subjectEcosystem:
			gfv.EcosystemId = sub.ecosystemID
		case subjectCorporation:
			gfv.CorporationId = sub.corporationID
		}
		if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return nil, fmt.Errorf("persist gfv: %w", err)
		}
		// Index by owner; network versions are listed in NetworkGovernance.
		switch sub.kind {
		case subjectEcosystem:
			err = ms.GFVersionByEcosystem.Set(ctx, collections.Join(sub.ecosystemID, msg.Version), gfv.Id)
		case subjectCorporation:
			err = ms.GFVersionByCorporation.Set(ctx, collections.Join(sub.corporationID, msg.Version), gfv.Id)
		case subjectNetwork:
			ng.NetworkGovernanceFrameworkVersions = append(ng.NetworkGovernanceFrameworkVersions, gfv.Id)
			err = ms.NetworkGovernance.Set(ctx, ng)
		}
		if err != nil {
			return nil, fmt.Errorf("index gfv: %w", err)
		}
	}

	// Upsert the document for (gfv, language) via the (gfv_id, language) index.
	idxKey := collections.Join(gfv.Id, msg.DocLanguage)
	var gfd types.GovernanceFrameworkDocument
	existingID, err := ms.GFDocumentByGFVLang.Get(ctx, idxKey)
	switch {
	case err == nil:
		gfd, err = ms.GFDocument.Get(ctx, existingID)
		if err != nil {
			return nil, fmt.Errorf("load gfd %d: %w", existingID, err)
		}
		gfd.Url = msg.DocUrl
		gfd.DigestSri = msg.DocDigestSri
	case errors.Is(err, collections.ErrNotFound):
		nextID, err := ms.GetNextID(ctx, "gfd")
		if err != nil {
			return nil, err
		}
		gfd = types.GovernanceFrameworkDocument{
			Id:        nextID,
			GfvId:     gfv.Id,
			Created:   ctx.BlockTime(),
			Language:  msg.DocLanguage,
			Url:       msg.DocUrl,
			DigestSri: msg.DocDigestSri,
		}
		if err := ms.GFDocumentByGFVLang.Set(ctx, idxKey, gfd.Id); err != nil {
			return nil, fmt.Errorf("index gfd: %w", err)
		}
	default:
		return nil, fmt.Errorf("lookup gfd: %w", err)
	}
	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return nil, fmt.Errorf("persist gfd: %w", err)
	}

	ev := events.New(types.ModuleName, types.EventTypeAddGovernanceFrameworkDocument).
		Uint(types.AttributeKeyGFVersionID, gfv.Id).
		Uint(types.AttributeKeyGFDocID, gfd.Id)
	if sub.kind == subjectNetwork {
		if err := ev.Msg(ms.cdc, msg); err != nil {
			return nil, err
		}
	} else {
		ev.Uint(events.AttrCorporationID, coView.Id)
	}
	ev.Emit(ctx)

	return &types.MsgAddGovernanceFrameworkDocumentResponse{}, nil
}

// networkTarget returns the network version a document for version goes to:
// the last listed version while it is a draft, else a new one after it.
func (k Keeper) networkTarget(ctx context.Context, version uint32) (types.GovernanceFrameworkVersion, bool, types.NetworkGovernance, error) {
	last, ng, err := k.lastNetworkVersion(ctx)
	if err != nil {
		return types.GovernanceFrameworkVersion{}, false, ng, err
	}
	if last.ActiveSince == nil {
		if version != last.Version {
			return types.GovernanceFrameworkVersion{}, false, ng, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d, the draft of the network governance framework", last.Version)
		}
		return last, true, ng, nil
	}
	if version != last.Version+1 {
		return types.GovernanceFrameworkVersion{}, false, ng, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d", last.Version+1)
	}
	return types.GovernanceFrameworkVersion{}, false, ng, nil
}
