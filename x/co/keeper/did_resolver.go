package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana-node/x/co/types"
)

// didOwnerRef is a shared container so setters propagate to by-value Keeper copies.
type didOwnerRef struct{ R types.DIDOwnerResolver }

// stubDIDOwnerResolver is the permissive default until app wiring injects the real one.
type stubDIDOwnerResolver struct{}

func (stubDIDOwnerResolver) ResolveDIDOwner(context.Context, string) (uint64, bool, error) {
	return 0, false, nil
}

func (k Keeper) SetEcosystemDIDResolver(r types.DIDOwnerResolver)   { k.ecDIDRef.R = r }
func (k Keeper) SetParticipantDIDResolver(r types.DIDOwnerResolver) { k.ppDIDRef.R = r }

func (k Keeper) ecosystemDIDResolver() types.DIDOwnerResolver   { return k.ecDIDRef.R }
func (k Keeper) participantDIDResolver() types.DIDOwnerResolver { return k.ppDIDRef.R }

// ResolveDIDOwner returns the corporation whose own `did` equals `did`.
// Corporation DIDs are globally unique, so the result is unambiguous.
func (k Keeper) ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error) {
	id, err := k.CorporationByDID.Get(ctx, did)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return id, true, nil
}

// assertNoForeignDIDOwner aborts if any Ecosystem or Participant binds `did`
// under a corporation != selfCorpID. selfCorpID=0 at create time (ids start at 1).
func (k Keeper) assertNoForeignDIDOwner(ctx context.Context, did string, selfCorpID uint64) error {
	if did == "" {
		return nil
	}
	for _, r := range []types.DIDOwnerResolver{k.ecosystemDIDResolver(), k.participantDIDResolver()} {
		owner, found, err := r.ResolveDIDOwner(ctx, did)
		if err != nil {
			return err
		}
		if found && owner != selfCorpID {
			return errorsmod.Wrapf(types.ErrDIDOwnershipConflict,
				"did %q is owned by corporation %d", did, owner)
		}
	}
	return nil
}
