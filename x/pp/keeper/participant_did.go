package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/verana-labs/verana-node/x/pp/types"
)

// assertDIDCorporationConsistent enforces the per-Participant
// (did, corporation_id) consistency invariant required at create time by
// spec [MOD-PP-MSG-1-2-1], [MOD-PP-MSG-7-2-1] and [MOD-PP-MSG-14-2-1]:
// every existing Participant entry that shares the given did MUST belong to
// corporationID; otherwise the create MUST abort. This is NOT a DID-uniqueness
// check — the same did may be reused across multiple participants of the same
// corporation. Backed by the ParticipantByDIDCorp index (see ResolveDIDOwner).
func (k Keeper) assertDIDCorporationConsistent(ctx context.Context, did string, corporationID uint64) error {
	if did == "" {
		return nil
	}
	owner, found, err := k.ResolveDIDOwner(ctx, did)
	if err != nil {
		return err
	}
	if found && owner != corporationID {
		return errorsmod.Wrapf(types.ErrDIDOwnershipConflict,
			"did %q is controlled by corporation %d, not %d", did, owner, corporationID)
	}
	return nil
}
