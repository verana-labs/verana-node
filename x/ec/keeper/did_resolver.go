package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana-node/x/ec/types"
)

// participantDIDRef is a shared container so setters propagate to by-value Keeper copies.
type participantDIDRef struct{ R types.DIDOwnerResolver }

// stubDIDOwnerResolver is the permissive default until app wiring injects the real one.
type stubDIDOwnerResolver struct{}

func (stubDIDOwnerResolver) ResolveDIDOwner(context.Context, string) (uint64, bool, error) {
	return 0, false, nil
}

func (k Keeper) SetParticipantDIDResolver(r types.DIDOwnerResolver) { k.ppDIDRef.R = r }

func (k Keeper) participantDIDResolver() types.DIDOwnerResolver { return k.ppDIDRef.R }

// ResolveDIDOwner returns the corporation controlling ecosystems with `did`.
// Per-ecosystem consistency guarantees a single owner. found=false if none.
func (k Keeper) ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error) {
	rng := collections.NewPrefixedPairRange[string, uint64](did)
	iter, err := k.EcosystemByDIDCorp.Iterate(ctx, rng)
	if err != nil {
		return 0, false, err
	}
	defer iter.Close()
	if iter.Valid() {
		corpID, err := iter.Value()
		if err != nil {
			return 0, false, err
		}
		return corpID, true, nil
	}
	return 0, false, nil
}

// assertNoForeignDIDOwnerES aborts if a Participant (other module) or the
// Corporation.did owner binds `did` under a corporation other than selfCorpID.
// Sibling ecosystems are already covered by assertDIDConsistent.
func (k Keeper) assertNoForeignDIDOwnerES(ctx context.Context, did string, selfCorpID uint64) error {
	if did == "" {
		return nil
	}
	// Participant side (cross-module, may be stubbed until wired).
	if owner, found, err := k.participantDIDResolver().ResolveDIDOwner(ctx, did); err != nil {
		return err
	} else if found && owner != selfCorpID {
		return errorsmod.Wrapf(types.ErrDIDOwnershipConflict, "did %q owned by corporation %d (participant)", did, owner)
	}
	// Corporation.did side (via the existing co keeper interface).
	if owner, found, err := k.coKeeper.ResolveDIDOwner(ctx, did); err != nil {
		return err
	} else if found && owner != selfCorpID {
		return errorsmod.Wrapf(types.ErrDIDOwnershipConflict, "did %q owned by corporation %d (corporation)", did, owner)
	}
	return nil
}
