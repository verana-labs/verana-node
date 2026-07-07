package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/ec/types"
)

// DIDConsistencyInvariant asserts that every Ecosystem sharing a did is
// controlled by the same corporation (MOD-ES-MSG-1-2-1).
func DIDConsistencyInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		didCorp := map[string]uint64{}
		var msg string
		broken := false
		_ = k.EcosystemByDIDCorp.Walk(ctx, nil, func(key collections.Pair[string, uint64], corpID uint64) (bool, error) {
			did := key.K1()
			if existing, ok := didCorp[did]; ok && existing != corpID {
				broken = true
				msg += fmt.Sprintf("did %q controlled by corporations %d and %d\n", did, existing, corpID)
			} else {
				didCorp[did] = corpID
			}
			return false, nil
		})
		return sdk.FormatInvariant(types.ModuleName, "did-consistency", msg), broken
	}
}
