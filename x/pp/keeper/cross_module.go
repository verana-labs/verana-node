package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cokeeper "github.com/verana-labs/verana-node/x/co/keeper"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// CoAsParticipantCorporationKeeper adapts x/co keeper to participanttypes.CorporationKeeper.
type CoAsParticipantCorporationKeeper struct {
	k cokeeper.Keeper
}

func NewCoAsParticipantCorporationKeeper(k cokeeper.Keeper) types.CorporationKeeper {
	return CoAsParticipantCorporationKeeper{k: k}
}

func (a CoAsParticipantCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	coID, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{Id: coID, PolicyAddress: policyAddress}, true
}

func (a CoAsParticipantCorporationKeeper) ResolveByID(ctx context.Context, id uint64) (types.CorporationView, bool) {
	co, err := a.k.Corporation.Get(ctx, id)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{Id: id, PolicyAddress: co.PolicyAddress}, true
}

func (a CoAsParticipantCorporationKeeper) ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error) {
	return a.k.ResolveDIDOwner(ctx, did)
}

// PpAsDIDOwnerResolver adapts the pp keeper for ec/co SetParticipantDIDResolver.
type PpAsDIDOwnerResolver struct{ k Keeper }

func NewPpAsDIDOwnerResolver(k Keeper) PpAsDIDOwnerResolver { return PpAsDIDOwnerResolver{k: k} }

func (a PpAsDIDOwnerResolver) ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error) {
	return a.k.ResolveDIDOwner(ctx, did)
}

// PpAsDeParticipantKeeper adapts the pp keeper for MOD-DE's participant view
// (AUTHZ-CHECK-3 step 1, MOD-DE-MSG-5-5). Wired via deKeeper.SetParticipantKeeper.
type PpAsDeParticipantKeeper struct{ k Keeper }

func NewPpAsDeParticipantKeeper(k Keeper) PpAsDeParticipantKeeper {
	return PpAsDeParticipantKeeper{k: k}
}

func (a PpAsDeParticipantKeeper) ViewParticipant(ctx context.Context, id uint64) (detypes.ParticipantView, bool) {
	p, err := a.k.Participant.Get(ctx, id)
	if err != nil {
		return detypes.ParticipantView{}, false
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	return detypes.ParticipantView{
		Active:         isActiveParticipant(p, now),
		Future:         isFutureParticipant(p, now),
		EffectiveUntil: p.EffectiveUntil,
	}, true
}
