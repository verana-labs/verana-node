package keeper

import (
	"context"
	"time"

	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// CheckNetworkGFAcceptance runs GF-CHECK-1 for co. No network governance
// framework version is in force yet, so every corporation passes.
func (k Keeper) CheckNetworkGFAcceptance(_ context.Context, _ types.CorporationView, _ time.Time) error {
	return nil
}

// CheckDelegateCGFAcceptance runs GF-CHECK-2 for a delegate acting under authz.
// Delegate acceptances are not recorded yet, so every delegate passes.
func (k Keeper) CheckDelegateCGFAcceptance(_ context.Context, _ uint64, _ *detypes.DelegateAuthorization, _ time.Time) error {
	return nil
}
