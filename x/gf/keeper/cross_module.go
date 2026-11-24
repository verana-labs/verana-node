package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// CreateInitialGFVersionForCorporation seeds version 1 of the Corporation's
// Governance Framework: one GovernanceFrameworkVersion + one
// GovernanceFrameworkDocument (in the Corporation's primary `language`).
//
// Called by MOD-CO MSG-1 (CreateCorporation) immediately after the Corporation
// row is persisted. There is no on-chain signer at this layer (keeper-to-keeper
// call), so AUTHZ is skipped — the parent MOD-CO MSG-1 has already validated
// the inputs via MsgCreateCorporation.ValidateBasic.
//
// Idempotency: aborts if (corpID, v=1) is already indexed. That can only
// happen on a buggy double-call from MOD-CO; it is NOT a legitimate retry path.
func (k Keeper) CreateInitialGFVersionForCorporation(
	ctx context.Context,
	corpID uint64,
	language, docURL, docDigestSRI string,
) error {
	if corpID == 0 {
		return cerrors.Wrap(types.ErrInvalidSubject, "corporation_id must be > 0")
	}
	if has, err := k.GFVersionByCorporation.Has(ctx, collections.Join(corpID, uint32(1))); err != nil {
		return fmt.Errorf("check existing gfv: %w", err)
	} else if has {
		return cerrors.Wrapf(types.ErrInvalidVersion, "initial GFV for corporation %d already exists", corpID)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	gfvID, err := k.GetNextID(sdkCtx, "gfv")
	if err != nil {
		return err
	}
	gfv := types.GovernanceFrameworkVersion{
		Id:            gfvID,
		CorporationId: corpID,
		Created:       now,
		Version:       1,
		ActiveSince:   &now, // MOD-CO MSG-1 seed: v1 is born active per spec.
	}
	if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("persist gfv: %w", err)
	}
	if err := k.GFVersionByCorporation.Set(ctx, collections.Join(corpID, uint32(1)), gfv.Id); err != nil {
		return fmt.Errorf("persist gfv corp index: %w", err)
	}

	gfdID, err := k.GetNextID(sdkCtx, "gfd")
	if err != nil {
		return err
	}
	gfd := types.GovernanceFrameworkDocument{
		Id:        gfdID,
		GfvId:     gfv.Id,
		Created:   now,
		Language:  language,
		Url:       docURL,
		DigestSri: docDigestSRI,
	}
	if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("persist gfd: %w", err)
	}
	if err := k.GFDocumentByGFVLang.Set(ctx, collections.Join(gfv.Id, language), gfd.Id); err != nil {
		return fmt.Errorf("persist gfd index: %w", err)
	}

	events.New(types.ModuleName, types.EventTypeAddGovernanceFrameworkDocument).
		Uint(events.AttrCorporationID, corpID).
		Uint(types.AttributeKeyGFVersionID, gfv.Id).
		Uint(types.AttributeKeyGFDocID, gfd.Id).
		Uint(types.AttributeKeyVersion, 1).
		Emit(sdkCtx)

	return nil
}

// CreateInitialGFVersionForEcosystem seeds version 1 of the Ecosystem's
// Governance Framework: one GovernanceFrameworkVersion + one
// GovernanceFrameworkDocument (in the Ecosystem's primary `language`).
//
// Called by MOD-ES MSG-1 (CreateEcosystem) immediately after the Ecosystem
// row is persisted. Mirror of CreateInitialGFVersionForCorporation; the
// resulting GFV is the ecosystem-owned half of the XOR (EcosystemId set,
// CorporationId zero), enforced by x/gf genesis at-rest invariants.
//
// AUTHZ is skipped (keeper-to-keeper). Idempotency: aborts if (ecID, v=1) is
// already indexed.
func (k Keeper) CreateInitialGFVersionForEcosystem(
	ctx context.Context,
	ecID uint64,
	language, docURL, docDigestSRI string,
) error {
	if ecID == 0 {
		return cerrors.Wrap(types.ErrInvalidSubject, "ecosystem_id must be > 0")
	}
	if has, err := k.GFVersionByEcosystem.Has(ctx, collections.Join(ecID, uint32(1))); err != nil {
		return fmt.Errorf("check existing gfv: %w", err)
	} else if has {
		return cerrors.Wrapf(types.ErrInvalidVersion, "initial GFV for ecosystem %d already exists", ecID)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	gfvID, err := k.GetNextID(sdkCtx, "gfv")
	if err != nil {
		return err
	}
	gfv := types.GovernanceFrameworkVersion{
		Id:          gfvID,
		EcosystemId: ecID,
		Created:     now,
		Version:     1,
		ActiveSince: &now, // MOD-ES MSG-1 seed: v1 is born active per spec.
	}
	if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("persist gfv: %w", err)
	}
	if err := k.GFVersionByEcosystem.Set(ctx, collections.Join(ecID, uint32(1)), gfv.Id); err != nil {
		return fmt.Errorf("persist gfv eco index: %w", err)
	}

	gfdID, err := k.GetNextID(sdkCtx, "gfd")
	if err != nil {
		return err
	}
	gfd := types.GovernanceFrameworkDocument{
		Id:        gfdID,
		GfvId:     gfv.Id,
		Created:   now,
		Language:  language,
		Url:       docURL,
		DigestSri: docDigestSRI,
	}
	if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("persist gfd: %w", err)
	}
	if err := k.GFDocumentByGFVLang.Set(ctx, collections.Join(gfv.Id, language), gfd.Id); err != nil {
		return fmt.Errorf("persist gfd index: %w", err)
	}

	events.New(types.ModuleName, types.EventTypeAddGovernanceFrameworkDocument).
		Uint(events.AttrEcosystemID, ecID).
		Uint(types.AttributeKeyGFVersionID, gfv.Id).
		Uint(types.AttributeKeyGFDocID, gfd.Id).
		Uint(types.AttributeKeyVersion, 1).
		Emit(sdkCtx)

	return nil
}

// ListVersionsByCorporation returns the Corporation's versions with their
// documents for MOD-CO-QRY-1. activeVersion and language are the corporation's;
// with activeOnly only the active version is returned, and preferredLang
// selects one document per version as collectDocs does.
func (k Keeper) ListVersionsByCorporation(ctx context.Context, corpID uint64, language string, activeVersion uint32, activeOnly bool, preferredLang string) ([]types.GovernanceFrameworkVersionWithDocs, error) {
	if corpID == 0 {
		return nil, cerrors.Wrap(types.ErrInvalidSubject, "corporation_id must be > 0")
	}
	return k.listVersions(ctx, k.GFVersionByCorporation, corpID, language, activeVersion, activeOnly, preferredLang)
}

// ListVersionsByEcosystem mirrors ListVersionsByCorporation for MOD-ES-QRY-1.
func (k Keeper) ListVersionsByEcosystem(ctx context.Context, ecID uint64, language string, activeVersion uint32, activeOnly bool, preferredLang string) ([]types.GovernanceFrameworkVersionWithDocs, error) {
	if ecID == 0 {
		return nil, cerrors.Wrap(types.ErrInvalidSubject, "ecosystem_id must be > 0")
	}
	return k.listVersions(ctx, k.GFVersionByEcosystem, ecID, language, activeVersion, activeOnly, preferredLang)
}

func (k Keeper) listVersions(ctx context.Context, index collections.Map[collections.Pair[uint64, uint32], uint64], ownerID uint64, language string, activeVersion uint32, activeOnly bool, preferredLang string) ([]types.GovernanceFrameworkVersionWithDocs, error) {
	var out []types.GovernanceFrameworkVersionWithDocs
	err := index.Walk(ctx, collections.NewPrefixedPairRange[uint64, uint32](ownerID), func(key collections.Pair[uint64, uint32], id uint64) (bool, error) {
		if activeOnly && key.K2() != activeVersion {
			return false, nil
		}
		gfv, err := k.GFVersion.Get(ctx, id)
		if err != nil {
			return true, fmt.Errorf("fetch gfv %d: %w", id, err)
		}
		docs, err := k.collectDocs(ctx, gfv.Id, preferredLang, language)
		if err != nil {
			return true, fmt.Errorf("collect docs: %w", err)
		}
		out = append(out, types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		})
		return false, nil
	})
	return out, err
}
