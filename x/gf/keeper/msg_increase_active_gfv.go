package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// IncreaseActiveGovernanceFrameworkVersion implements MOD-GF-MSG-2.
func (ms msgServer) IncreaseActiveGovernanceFrameworkVersion(goCtx context.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) (*types.MsgIncreaseActiveGovernanceFrameworkVersionResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Authority != "" {
		return ms.increaseActiveNetworkVersion(ctx, msg)
	}

	coView, err := ms.authorize(ctx, msg.Corporation, msg.Delegate, detypes.MsgType_MSG_TYPE_INCREASE_ACTIVE_GOVERNANCE_FRAMEWORK_VERSION)
	if err != nil {
		return nil, err
	}

	sub, err := ms.resolveSubject(ctx, coView, msg.EcosystemId)
	if err != nil {
		return nil, err
	}

	nextVersion := sub.activeVersion + 1
	var gfvID uint64
	switch sub.kind {
	case subjectEcosystem:
		gfvID, err = ms.GFVersionByEcosystem.Get(ctx, collections.Join(sub.ecosystemID, nextVersion))
	case subjectCorporation:
		gfvID, err = ms.GFVersionByCorporation.Get(ctx, collections.Join(sub.corporationID, nextVersion))
	}
	if err != nil {
		return nil, cerrors.Wrapf(types.ErrNoActivatableVersion, "no GFV for next version %d", nextVersion)
	}
	gfv, err := ms.GFVersion.Get(ctx, gfvID)
	if err != nil {
		return nil, fmt.Errorf("fetch gfv %d: %w", gfvID, err)
	}

	// A document for subject.language MUST exist on this version.
	hasDefaultLang, err := ms.GFDocumentByGFVLang.Has(ctx, collections.Join(gfv.Id, sub.language))
	if err != nil {
		return nil, fmt.Errorf("lookup gfd: %w", err)
	}
	if !hasDefaultLang {
		return nil, types.ErrMissingDefaultLang
	}
	if err := ms.checkAcceptanceGrace(ctx, msg.AcceptanceGrace); err != nil {
		return nil, err
	}

	now := ctx.BlockTime()
	deadline := now.Add(msg.AcceptanceGrace)
	gfv.ActiveSince = &now
	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return nil, fmt.Errorf("persist gfv: %w", err)
	}

	ev := events.New(types.ModuleName, types.EventTypeIncreaseActiveGovernanceFrameworkVersion).
		Uint(events.AttrCorporationID, coView.Id)
	switch sub.kind {
	case subjectEcosystem:
		if err := ms.ecosystemKeeper().SetEcosystemActiveVersion(ctx, sub.ecosystemID, nextVersion, deadline); err != nil {
			return nil, fmt.Errorf("update ecosystem active version: %w", err)
		}
		ev.Uint(events.AttrEcosystemID, sub.ecosystemID)
	case subjectCorporation:
		if err := ms.corporationKeeper().SetActiveVersion(ctx, sub.corporationID, nextVersion, deadline); err != nil {
			return nil, fmt.Errorf("update corporation active version: %w", err)
		}
	}
	ev.Uint(types.AttributeKeyGFVersionID, gfv.Id).
		Uint(types.AttributeKeyVersion, uint64(nextVersion)).
		Time(types.AttributeKeyActiveSince, now).
		Time(types.AttributeKeyAcceptanceDeadline, deadline).
		Emit(ctx)

	return &types.MsgIncreaseActiveGovernanceFrameworkVersionResponse{}, nil
}

// increaseActiveNetworkVersion activates the draft of the network governance
// framework and opens its acceptance grace period.
func (ms msgServer) increaseActiveNetworkVersion(ctx sdk.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) (*types.MsgIncreaseActiveGovernanceFrameworkVersionResponse, error) {
	if err := ms.checkAuthority(msg.Authority); err != nil {
		return nil, err
	}
	gfv, ng, err := ms.lastNetworkVersion(ctx)
	if err != nil {
		return nil, err
	}
	if gfv.ActiveSince != nil {
		return nil, cerrors.Wrapf(types.ErrNoActivatableVersion, "network governance framework version %d is already active", gfv.Version)
	}
	hasDoc := false
	if err := ms.GFDocumentByGFVLang.Walk(ctx, collections.NewPrefixedPairRange[uint64, string](gfv.Id), func(collections.Pair[uint64, string], uint64) (bool, error) {
		hasDoc = true
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("lookup gfd: %w", err)
	}
	if !hasDoc {
		return nil, cerrors.Wrapf(types.ErrMissingDefaultLang, "network governance framework version %d has no document", gfv.Version)
	}
	if err := ms.checkAcceptanceGrace(ctx, msg.AcceptanceGrace); err != nil {
		return nil, err
	}

	now := ctx.BlockTime()
	deadline := now.Add(msg.AcceptanceGrace)
	gfv.ActiveSince = &now
	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return nil, fmt.Errorf("persist gfv: %w", err)
	}
	ng.NgfAcceptanceDeadline = &deadline
	if err := ms.NetworkGovernance.Set(ctx, ng); err != nil {
		return nil, fmt.Errorf("persist network governance: %w", err)
	}

	ev := events.New(types.ModuleName, types.EventTypeIncreaseActiveGovernanceFrameworkVersion).
		Bool(types.AttributeKeyNetwork, true).
		Uint(types.AttributeKeyGFVersionID, gfv.Id).
		Uint(types.AttributeKeyVersion, uint64(gfv.Version)).
		Time(types.AttributeKeyActiveSince, now).
		Time(types.AttributeKeyAcceptanceDeadline, deadline)
	if err := ev.Msg(ms.cdc, msg); err != nil {
		return nil, err
	}
	ev.Emit(ctx)

	return &types.MsgIncreaseActiveGovernanceFrameworkVersionResponse{}, nil
}

func (ms msgServer) checkAcceptanceGrace(ctx sdk.Context, grace time.Duration) error {
	if min := ms.GetParams(ctx).GfAcceptanceGraceMin; grace < min {
		return cerrors.Wrapf(types.ErrAcceptanceGraceTooShort, "%s < %s", grace, min)
	}
	return nil
}
