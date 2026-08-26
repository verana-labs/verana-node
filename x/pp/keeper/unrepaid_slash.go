package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/pp/types"
)

// [MOD-PP-MSG-1-2-5] / [MOD-PP-MSG-2-2-4] / [MOD-PP-MSG-14-2-5] unrepaid slash
// checks: a corporation with an unrepaid ecosystem slash in ecosystemId, or an
// unrepaid network slash on its trust deposit, cannot obtain a new Participant.
// The amount comparison (not repaid == nil) stays correct after a repay and a
// second slash.
func (ms msgServer) checkUnrepaidSlash(ctx sdk.Context, corporationId uint64, ecosystemId uint64) error {
	var blockedID uint64
	var found bool
	err := ms.Participant.Walk(ctx, nil, func(_ uint64, p types.Participant) (bool, error) {
		if p.CorporationId != corporationId || p.Slashed == nil || p.RepaidDeposit >= p.SlashedDeposit {
			return false, nil
		}
		cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, p.SchemaId)
		if err != nil {
			return true, fmt.Errorf("credential schema not found for participant %d: %w", p.Id, err)
		}
		if cs.EcosystemId != ecosystemId {
			return false, nil
		}
		found = true
		blockedID = p.Id
		return true, nil
	})
	if err != nil {
		return err
	}
	if found {
		return fmt.Errorf("corporation has an unrepaid slash on participant %d in this ecosystem: repay it via RepayParticipantSlashedTrustDeposit first", blockedID)
	}
	hasNetworkSlash, err := ms.trustDeposit.HasUnrepaidSlash(ctx, corporationId)
	if err != nil {
		return err
	}
	if hasNetworkSlash {
		return fmt.Errorf("corporation trust deposit has an unrepaid slash: repay it via RepaySlashedTrustDeposit first")
	}
	return nil
}
