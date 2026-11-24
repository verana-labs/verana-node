package keeper

import (
	"context"

	"github.com/verana-labs/verana-node/x/de/types"
)

// SetDelegateAcceptedCGFVersion records that delegate accepted version of the
// governance framework of corporationID (MOD-GF-MSG-4-3).
func (k Keeper) SetDelegateAcceptedCGFVersion(ctx context.Context, corporationID uint64, delegate string, version uint32) error {
	da, found, err := k.getDelegateAuthorizationByCorpDelegate(ctx, corporationID, delegate)
	if err != nil {
		return err
	}
	if !found {
		return types.ErrAuthzNotFound
	}
	da.AcceptedCgfVersion = version
	return k.DelegateAuthorizations.Set(ctx, da.Id, da)
}
