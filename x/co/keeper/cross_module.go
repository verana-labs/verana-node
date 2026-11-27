package keeper

import (
	"context"
	stderrors "errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/co/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	gftypes "github.com/verana-labs/verana-node/x/gf/types"
)

// CoAsGFCorporationKeeper adapts the MOD-CO keeper to MOD-GF's CorporationKeeper
// interface. Wired post-construction via gfKeeper.SetCorporationKeeper to break
// the MOD-GF ↔ MOD-CO depinject cycle.
type CoAsGFCorporationKeeper struct {
	k Keeper
}

func NewCoAsGFCorporationKeeper(k Keeper) gftypes.CorporationKeeper {
	return CoAsGFCorporationKeeper{k: k}
}

// ResolveCorporationByPolicyAddress backs MOD-GF AUTHZ-CHECK-5.
func (a CoAsGFCorporationKeeper) ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (gftypes.CorporationView, error) {
	co, err := a.k.ResolveCorporationByPolicyAddress(ctx, policyAddress)
	if err != nil {
		return gftypes.CorporationView{}, err
	}
	return gfCorporationView(&co), nil
}

// ResolveSigner backs the MOD-GF group-proposal path.
func (a CoAsGFCorporationKeeper) ResolveSigner(ctx context.Context, corporation, signer string) (gftypes.CorporationView, bool, error) {
	co, group, err := a.k.ResolveSigner(ctx, corporation, signer)
	if err != nil {
		return gftypes.CorporationView{}, false, err
	}
	return gfCorporationView(&co), group, nil
}

func (a CoAsGFCorporationKeeper) GetByID(ctx context.Context, corporationID uint64) (gftypes.CorporationView, bool) {
	co, err := a.k.Corporation.Get(ctx, corporationID)
	if err != nil {
		return gftypes.CorporationView{}, false
	}
	return gfCorporationView(&co), true
}

func gfCorporationView(co *types.Corporation) gftypes.CorporationView {
	return gftypes.CorporationView{
		Id:            co.Id,
		PolicyAddress: co.PolicyAddress,
		Language:      co.Language,
		ActiveVersion: co.ActiveVersion,
	}
}

// CoAsDeCorporationKeeper adapts the MOD-CO keeper to MOD-DE's CorporationKeeper
// interface. Wired post-construction via deKeeper.SetCorporationKeeper to break
// the MOD-DE ↔ MOD-CO depinject cycle (MOD-CO depends on MOD-DE's
// DelegationKeeper for AUTHZ-CHECK-1).
type CoAsDeCorporationKeeper struct {
	k Keeper
}

func NewCoAsDeCorporationKeeper(k Keeper) detypes.CorporationKeeper {
	return CoAsDeCorporationKeeper{k: k}
}

// ResolveCorporationByPolicyAddress backs MOD-DE AUTHZ-CHECK-5, routing through
// the canonical resolver so an unregistered signer aborts with
// ErrCorporationNotRegistered (referencing MOD-CO-MSG-1).
func (a CoAsDeCorporationKeeper) ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (detypes.CorporationView, error) {
	co, err := a.k.ResolveCorporationByPolicyAddress(ctx, policyAddress)
	if err != nil {
		return detypes.CorporationView{}, err
	}
	return deCorporationView(&co), nil
}

// ResolveSigner backs the MOD-DE group-proposal path.
func (a CoAsDeCorporationKeeper) ResolveSigner(ctx context.Context, corporation, signer string) (detypes.CorporationView, bool, error) {
	co, group, err := a.k.ResolveSigner(ctx, corporation, signer)
	if err != nil {
		return detypes.CorporationView{}, false, err
	}
	return detypes.CorporationView{Id: co.Id, PolicyAddress: co.PolicyAddress}, group, nil
}

// ResolveCorporationByID resolves a corporation_id to its policy_address, used by
// MOD-DE-MSG-1 to set the x/feegrant granter.
func (a CoAsDeCorporationKeeper) ResolveCorporationByID(ctx context.Context, id uint64) (detypes.CorporationView, error) {
	co, err := a.k.Corporation.Get(ctx, id)
	if err != nil {
		return detypes.CorporationView{}, err
	}
	return deCorporationView(&co), nil
}

func (a CoAsDeCorporationKeeper) GetCorporationByPolicyAddress(ctx context.Context, policyAddress string) (detypes.CorporationView, bool, error) {
	id, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if stderrors.Is(err, collections.ErrNotFound) {
		return detypes.CorporationView{}, false, nil
	}
	if err != nil {
		return detypes.CorporationView{}, false, err
	}
	co, err := a.k.Corporation.Get(ctx, id)
	if err != nil {
		return detypes.CorporationView{}, false, err
	}
	return deCorporationView(&co), true, nil
}

// ClearSponsor is called by MOD-DE-MSG-11 (Revoke Sponsorship).
func (a CoAsDeCorporationKeeper) ClearSponsor(ctx context.Context, corporationID, sponsorID uint64) (bool, error) {
	co, err := a.k.Corporation.Get(ctx, corporationID)
	if err != nil {
		return false, fmt.Errorf("corporation %d not found: %w", corporationID, err)
	}
	if co.SponsorCorporationId != sponsorID {
		return false, nil
	}
	co.SponsorCorporationId = 0
	return true, a.k.Corporation.Set(ctx, co.Id, co)
}

func deCorporationView(co *types.Corporation) detypes.CorporationView {
	return detypes.CorporationView{Id: co.Id, PolicyAddress: co.PolicyAddress, SponsorCorporationId: co.SponsorCorporationId}
}

// SetActiveVersion is called by MOD-GF MSG-2 (IncreaseActiveGovernanceFrameworkVersion).
func (a CoAsGFCorporationKeeper) SetActiveVersion(ctx context.Context, corporationID uint64, newVersion uint32) error {
	co, err := a.k.Corporation.Get(ctx, corporationID)
	if err != nil {
		return fmt.Errorf("corporation %d not found: %w", corporationID, err)
	}
	co.ActiveVersion = newVersion
	co.Modified = sdk.UnwrapSDKContext(ctx).BlockTime()
	return a.k.Corporation.Set(ctx, co.Id, co)
}
