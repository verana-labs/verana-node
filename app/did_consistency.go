package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	ectypes "github.com/verana-labs/verana-node/x/ec/types"
	pptypes "github.com/verana-labs/verana-node/x/pp/types"
)

// assertGenesisDIDConsistency enforces the global DID ownership invariant across
// Corporation, Ecosystem and Participant at genesis import: every DID must
// resolve to a single corporation. Violations are rejected (not repaired) so a
// bad genesis is fixed by the operator, not silently mutated.
func (app *App) assertGenesisDIDConsistency(ctx sdk.Context) error {
	owner := map[string]uint64{}
	conflicts := map[string]struct{}{}
	record := func(did string, corp uint64) {
		if did == "" {
			return
		}
		if prev, ok := owner[did]; ok {
			if prev != corp {
				conflicts[did] = struct{}{}
			}
			return
		}
		owner[did] = corp
	}

	if err := app.CoKeeper.Corporation.Walk(ctx, nil, func(_ uint64, c cotypes.Corporation) (bool, error) {
		record(c.Did, c.Id)
		return false, nil
	}); err != nil {
		return err
	}
	if err := app.EcosystemKeeper.Ecosystem.Walk(ctx, nil, func(_ uint64, e ectypes.Ecosystem) (bool, error) {
		record(e.Did, e.CorporationId)
		return false, nil
	}); err != nil {
		return err
	}
	if err := app.ParticipantKeeper.Participant.Walk(ctx, nil, func(_ uint64, p pptypes.Participant) (bool, error) {
		record(p.Did, p.CorporationId)
		return false, nil
	}); err != nil {
		return err
	}

	if len(conflicts) > 0 {
		dids := make([]string, 0, len(conflicts))
		for did := range conflicts {
			dids = append(dids, did)
		}
		return fmt.Errorf("genesis violates DID ownership invariant for did(s): %v", dids)
	}
	return nil
}
