package keeper

import (
	"context"

	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/util/events"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// AcceptGovernanceFrameworkVersion implements MOD-GF-MSG-3.
func (ms msgServer) AcceptGovernanceFrameworkVersion(goCtx context.Context, msg *types.MsgAcceptGovernanceFrameworkVersion) (*types.MsgAcceptGovernanceFrameworkVersionResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// GF-CHECK-1 is skipped when the network framework is accepted.
	var (
		coView types.CorporationView
		err    error
	)
	if msg.Network {
		coView, err = ms.corporationKeeper().GetCorporationByPolicyAddress(ctx, msg.Corporation)
	} else {
		coView, err = ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, msg.Corporation)
	}
	if err != nil {
		return nil, err
	}
	groupPath := msg.Delegate == msg.Corporation
	if !groupPath {
		if err := ms.delegationKeeper.CheckDelegateAuthorization(ctx, coView.Id, msg.Delegate, detypes.MsgType_MSG_TYPE_ACCEPT_GOVERNANCE_FRAMEWORK_VERSION, now); err != nil {
			return nil, err
		}
	}

	ev := events.New(types.ModuleName, types.EventTypeAcceptGovernanceFrameworkVersion).
		Uint(events.AttrCorporationID, coView.Id)
	if msg.Network {
		ngf, _, err := ms.networkVersionInForce(ctx)
		if err != nil {
			return nil, err
		}
		if msg.Version != ngf.Version {
			return nil, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d, the network governance framework version in force", ngf.Version)
		}
		if err := ms.corporationKeeper().SetAcceptedNgfVersion(ctx, coView.Id, msg.Version); err != nil {
			return nil, err
		}
		ev.Bool(types.AttributeKeyNetwork, true)
	} else {
		e, ok := ms.ecosystemKeeper().GetEcosystemView(ctx, msg.EcosystemId)
		if !ok {
			return nil, cerrors.Wrapf(types.ErrSubjectNotFound, "ecosystem %d", msg.EcosystemId)
		}
		if e.CorporationID == coView.Id {
			return nil, types.ErrControllerAcceptance
		}
		if msg.Version != e.ActiveVersion {
			return nil, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d, the active version of ecosystem %d", e.ActiveVersion, e.Id)
		}
		if err := ms.setEcosystemAcceptance(ctx, types.EcosystemAcceptance{
			EcosystemId:   e.Id,
			CorporationId: coView.Id,
			Version:       msg.Version,
			Accepted:      now,
		}); err != nil {
			return nil, err
		}
		ev.Uint(events.AttrEcosystemID, e.Id)
	}
	ev.Uint(types.AttributeKeyVersion, uint64(msg.Version)).Time(events.AttrNow, now)
	if groupPath {
		if err := ev.Msg(ms.cdc, msg); err != nil {
			return nil, err
		}
	}
	ev.Emit(ctx)

	return &types.MsgAcceptGovernanceFrameworkVersionResponse{}, nil
}

// AcceptCorporationGovernanceFramework implements MOD-GF-MSG-4. It runs outside
// AUTHZ-CHECK: accepting is what makes the delegate's authorization usable.
func (ms msgServer) AcceptCorporationGovernanceFramework(goCtx context.Context, msg *types.MsgAcceptCorporationGovernanceFramework) (*types.MsgAcceptCorporationGovernanceFrameworkResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	co, ok := ms.corporationKeeper().GetByID(ctx, msg.CorporationId)
	if !ok {
		return nil, cerrors.Wrapf(types.ErrSubjectNotFound, "corporation %d", msg.CorporationId)
	}
	if msg.Version != co.ActiveVersion {
		return nil, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d, the active version of corporation %d", co.ActiveVersion, co.Id)
	}
	if err := ms.delegationKeeper.SetDelegateAcceptedCGFVersion(ctx, co.Id, msg.Delegate, msg.Version); err != nil {
		return nil, err
	}

	events.New(types.ModuleName, types.EventTypeAcceptCorporationGovernanceFramework).
		Uint(events.AttrCorporationID, co.Id).
		Str(types.AttributeKeyDelegate, msg.Delegate).
		Uint(types.AttributeKeyVersion, uint64(msg.Version)).
		Time(events.AttrNow, ctx.BlockTime()).
		Emit(ctx)

	return &types.MsgAcceptCorporationGovernanceFrameworkResponse{}, nil
}
