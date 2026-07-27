package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana-node/x/pp/types"
)

// IndexParticipantDID writes the (did, id) -> corporation_id row. Participants
// are never deleted and did/corp are immutable, so the index is create-only.
func (k Keeper) IndexParticipantDID(ctx context.Context, p types.Participant) error {
	if p.Did == "" {
		return nil
	}
	return k.ParticipantByDIDCorp.Set(ctx, collections.Join(p.Did, p.Id), p.CorporationId)
}

// ResolveDIDOwner returns the corporation owning a participant with `did`.
// Per-participant consistency guarantees a single owner. found=false if none.
func (k Keeper) ResolveDIDOwner(ctx context.Context, did string) (uint64, bool, error) {
	rng := collections.NewPrefixedPairRange[string, uint64](did)
	iter, err := k.ParticipantByDIDCorp.Iterate(ctx, rng)
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

// assertNoForeignDIDOwnerPP aborts if an Ecosystem or the Corporation.did owner
// binds `did` under a corporation != corporationID. Sibling participants are
// covered by assertDIDCorporationConsistent.
func (k Keeper) assertNoForeignDIDOwnerPP(ctx context.Context, did string, corporationID uint64) error {
	if did == "" {
		return nil
	}
	if owner, found, err := k.ecosystemKeeper.ResolveDIDOwner(ctx, did); err != nil {
		return err
	} else if found && owner != corporationID {
		return errorsmod.Wrapf(types.ErrDIDOwnershipConflict, "did %q owned by corporation %d (ecosystem)", did, owner)
	}
	if owner, found, err := k.coKeeper.ResolveDIDOwner(ctx, did); err != nil {
		return err
	} else if found && owner != corporationID {
		return errorsmod.Wrapf(types.ErrDIDOwnershipConflict, "did %q owned by corporation %d (corporation)", did, owner)
	}
	return nil
}
