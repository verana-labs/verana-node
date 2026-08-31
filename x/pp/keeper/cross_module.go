package keeper

import (
	"context"

	cokeeper "github.com/verana-labs/verana-node/x/co/keeper"
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
