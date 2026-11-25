package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/cs/types"
)

// checkSchemaOwnership enforces that the signing corporation, resolved by
// AUTHZ-CHECK-5 to corporationID, controls the Ecosystem that owns cs.
func (k Keeper) checkSchemaOwnership(ctx sdk.Context, cs types.CredentialSchema, corporationID uint64) error {
	ec, err := k.ecosystemKeeper.GetEcosystem(ctx, cs.EcosystemId)
	if err != nil {
		return fmt.Errorf("ecosystem %d not found: %w", cs.EcosystemId, err)
	}
	if ec.CorporationId != corporationID {
		return fmt.Errorf("corporation %d does not control the ecosystem (%d) that owns this credential schema", corporationID, cs.EcosystemId)
	}
	return nil
}

// checkCreateSchemaOwnership enforces ownership at CreateCredentialSchema time
// where the CredentialSchema entry does not yet exist.
func (k Keeper) checkCreateSchemaOwnership(ctx sdk.Context, ecosystemID, corporationID uint64) error {
	ec, err := k.ecosystemKeeper.GetEcosystem(ctx, ecosystemID)
	if err != nil {
		return fmt.Errorf("ecosystem %d not found: %w", ecosystemID, err)
	}
	if ec.CorporationId != corporationID {
		return fmt.Errorf("corporation %d does not control ecosystem %d", corporationID, ecosystemID)
	}
	return nil
}
