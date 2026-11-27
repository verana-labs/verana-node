package keeper

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// CheckNetworkGFAcceptance runs GF-CHECK-1 for co.
func (k Keeper) CheckNetworkGFAcceptance(ctx context.Context, co types.CorporationView, now time.Time) error {
	ngf, ng, err := k.networkVersionInForce(ctx)
	if err != nil {
		return err
	}
	if co.AcceptedNgfVersion == ngf.Version {
		return nil
	}
	if ng.NgfAcceptanceDeadline != nil && now.Before(*ng.NgfAcceptanceDeadline) {
		return nil
	}
	return errorsmod.Wrapf(types.ErrGFNotAccepted, "corporation %d must accept version %d of the network governance framework", co.Id, ngf.Version)
}

// CheckDelegateCGFAcceptance runs GF-CHECK-2 for a delegate acting under authz.
func (k Keeper) CheckDelegateCGFAcceptance(ctx context.Context, corporationID uint64, authz *detypes.DelegateAuthorization, now time.Time) error {
	co, ok := k.corporationKeeper().GetByID(ctx, corporationID)
	if !ok {
		return errorsmod.Wrapf(types.ErrSubjectNotFound, "corporation %d", corporationID)
	}
	if authz.AcceptedCgfVersion == co.ActiveVersion {
		return nil
	}
	if authz.AcceptedCgfVersion != 0 && co.AcceptanceDeadline != nil && now.Before(*co.AcceptanceDeadline) {
		return nil
	}
	return errorsmod.Wrapf(types.ErrGFNotAccepted, "delegate %s must accept version %d of the governance framework of corporation %d", authz.Delegate, co.ActiveVersion, co.Id)
}

// CheckEcosystemGFAcceptance runs GF-CHECK-3 for corporationID acting in ecosystemID.
func (k Keeper) CheckEcosystemGFAcceptance(ctx context.Context, ecosystemID, corporationID uint64, now time.Time) error {
	e, ok := k.ecosystemKeeper().GetEcosystemView(ctx, ecosystemID)
	if !ok {
		return errorsmod.Wrapf(types.ErrSubjectNotFound, "ecosystem %d", ecosystemID)
	}
	if e.CorporationID == corporationID {
		return nil
	}
	acc, found, err := k.getEcosystemAcceptance(ctx, ecosystemID, corporationID)
	if err != nil {
		return err
	}
	if found && acc.Version == e.ActiveVersion {
		return nil
	}
	if found && e.AcceptanceDeadline != nil && now.Before(*e.AcceptanceDeadline) {
		return nil
	}
	return errorsmod.Wrapf(types.ErrGFNotAccepted, "corporation %d must accept version %d of the governance framework of ecosystem %d", corporationID, e.ActiveVersion, ecosystemID)
}

// NetworkGFVersionInForce returns the version of the network governance
// framework in force.
func (k Keeper) NetworkGFVersionInForce(ctx context.Context) (uint32, error) {
	gfv, _, err := k.networkVersionInForce(ctx)
	if err != nil {
		return 0, err
	}
	return gfv.Version, nil
}

func (k Keeper) getNetworkGovernance(ctx context.Context) (types.NetworkGovernance, error) {
	ng, err := k.NetworkGovernance.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return types.NetworkGovernance{}, nil
	}
	return ng, err
}

// networkVersionInForce returns the last listed network version whose
// active_since is set, and the network governance row it read.
func (k Keeper) networkVersionInForce(ctx context.Context) (types.GovernanceFrameworkVersion, types.NetworkGovernance, error) {
	ng, err := k.getNetworkGovernance(ctx)
	if err != nil {
		return types.GovernanceFrameworkVersion{}, ng, err
	}
	ids := ng.NetworkGovernanceFrameworkVersions
	for i := len(ids) - 1; i >= 0; i-- {
		gfv, err := k.GFVersion.Get(ctx, ids[i])
		if err != nil {
			return types.GovernanceFrameworkVersion{}, ng, err
		}
		if gfv.ActiveSince != nil {
			return gfv, ng, nil
		}
	}
	return types.GovernanceFrameworkVersion{}, ng, errorsmod.Wrap(types.ErrSubjectNotFound, "no network governance framework version in force")
}

// lastNetworkVersion returns the network version with the highest version.
func (k Keeper) lastNetworkVersion(ctx context.Context) (types.GovernanceFrameworkVersion, types.NetworkGovernance, error) {
	ng, err := k.getNetworkGovernance(ctx)
	if err != nil {
		return types.GovernanceFrameworkVersion{}, ng, err
	}
	ids := ng.NetworkGovernanceFrameworkVersions
	if len(ids) == 0 {
		return types.GovernanceFrameworkVersion{}, ng, errorsmod.Wrap(types.ErrSubjectNotFound, "no network governance framework version")
	}
	gfv, err := k.GFVersion.Get(ctx, ids[len(ids)-1])
	return gfv, ng, err
}

// getEcosystemAcceptance returns the acceptance of (ecosystemID, corporationID)
// with its key fields set.
func (k Keeper) getEcosystemAcceptance(ctx context.Context, ecosystemID, corporationID uint64) (types.EcosystemAcceptance, bool, error) {
	acc, err := k.EcosystemAcceptances.Get(ctx, collections.Join(ecosystemID, corporationID))
	if errors.Is(err, collections.ErrNotFound) {
		return types.EcosystemAcceptance{}, false, nil
	}
	if err != nil {
		return types.EcosystemAcceptance{}, false, err
	}
	acc.EcosystemId, acc.CorporationId = ecosystemID, corporationID
	return acc, true, nil
}

// setEcosystemAcceptance stores acc without its key fields.
func (k Keeper) setEcosystemAcceptance(ctx context.Context, acc types.EcosystemAcceptance) error {
	key := collections.Join(acc.EcosystemId, acc.CorporationId)
	acc.EcosystemId, acc.CorporationId = 0, 0
	return k.EcosystemAcceptances.Set(ctx, key, acc)
}
