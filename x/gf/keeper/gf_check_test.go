package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// No network governance framework version and no delegate acceptance is
// recorded yet, so both checks pass.
func TestGFCheck_NothingRecordedPasses(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	require.NoError(t, k.CheckNetworkGFAcceptance(ctx, types.CorporationView{Id: 1, ActiveVersion: 1}, now))
	require.NoError(t, k.CheckDelegateCGFAcceptance(ctx, 1, &detypes.DelegateAuthorization{CorporationId: 1, Delegate: testDelegate}, now))
}
