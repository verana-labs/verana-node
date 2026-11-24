package trustdeposit_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/testutil/nullify"
	trustdeposit "github.com/verana-labs/verana-node/x/td/module"
	"github.com/verana-labs/verana-node/x/td/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:     types.DefaultParams(),
		TuPegValue: types.DefaultTuPegValueGenesis,
		TrustDeposits: []types.TrustDepositRecord{
			{
				CorporationId: 1,
				Share:         math.LegacyNewDec(100),
				Deposit:       1000,
				Refunded:      50,
			},
			{
				CorporationId: 2,
				Share:         math.LegacyNewDec(200),
				Deposit:       2000,
				Refunded:      100,
			},
		},
	}

	k, ctx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(ctx, k, genesisState)
	got := trustdeposit.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	// Use nullify to ignore fields that are updated by the keeper (timestamps, etc.)
	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// Verify params
	require.Equal(t, genesisState.Params, got.Params)

	// Verify trust deposits (may need to sort since map iteration is non-deterministic)
	require.ElementsMatch(t, genesisState.TrustDeposits, got.TrustDeposits)

	// Verify that trust deposits were correctly stored in the keeper
	for _, td := range genesisState.TrustDeposits {
		stored, err := k.TrustDeposit.Get(ctx, td.CorporationId)
		require.NoError(t, err)
		require.Equal(t, td.CorporationId, stored.CorporationId)
		require.Equal(t, td.Share, stored.Share)
		require.Equal(t, td.Deposit, stored.Deposit)
		require.Equal(t, td.Refunded, stored.Refunded)
	}
}

// TestGenesisRoundtripPreservesSlashState verifies cumulative slash state survives
// export/import so the slash-unpaid gate is not erased on a genesis round-trip.
func TestGenesisRoundtripPreservesSlashState(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	ts := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	td := types.TrustDeposit{
		CorporationId:  7,
		Share:          math.LegacyNewDec(500),
		Deposit:        500,
		Refunded:       20,
		SlashedDeposit: 300,
		RepaidDeposit:  100,
		LastSlashed:    &ts,
		SlashCount:     2,
	}
	require.NoError(t, k.TrustDeposit.Set(ctx, td.CorporationId, td))

	exported := trustdeposit.ExportGenesis(ctx, k)
	newK, newCtx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(newCtx, newK, *exported)

	got, err := newK.TrustDeposit.Get(newCtx, td.CorporationId)
	require.NoError(t, err)
	require.Equal(t, uint64(300), got.SlashedDeposit)
	require.Equal(t, uint64(100), got.RepaidDeposit)
	require.Equal(t, uint64(2), got.SlashCount)
	require.NotNil(t, got.LastSlashed)
	require.True(t, got.LastSlashed.Equal(ts))
}

// TestEmptyGenesis tests the initialization with empty genesis state
func TestEmptyGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:        types.DefaultParams(),
		TuPegValue:    types.DefaultTuPegValueGenesis,
		TrustDeposits: []types.TrustDepositRecord{},
	}

	k, ctx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(ctx, k, genesisState)
	exported := trustdeposit.ExportGenesis(ctx, k)

	require.Equal(t, genesisState.Params, exported.Params)
	require.Empty(t, exported.TrustDeposits)
}

// TestImportExportGenesisWithTrustDeposits tests that import/export preserves state
func TestImportExportGenesisWithTrustDeposits(t *testing.T) {
	// Create keeper and context
	k, ctx := keepertest.TrustdepositKeeper(t)

	// Create trust deposits directly with the keeper (keyed by corporation_id)
	td1 := types.TrustDeposit{
		CorporationId: 1,
		Share:         math.LegacyNewDec(100),
		Deposit:       1000,
		Refunded:      50,
	}

	td2 := types.TrustDeposit{
		CorporationId: 2,
		Share:         math.LegacyNewDec(200),
		Deposit:       2000,
		Refunded:      100,
	}

	// Save trust deposits
	require.NoError(t, k.TrustDeposit.Set(ctx, td1.CorporationId, td1))
	require.NoError(t, k.TrustDeposit.Set(ctx, td2.CorporationId, td2))

	// Export genesis
	exported := trustdeposit.ExportGenesis(ctx, k)

	// Verify exported genesis contains the trust deposits
	require.Len(t, exported.TrustDeposits, 2)

	// Initialize a new keeper with the exported genesis
	newK, newCtx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(newCtx, newK, *exported)

	// Verify the trust deposits were correctly imported
	storedTd1, err := newK.TrustDeposit.Get(newCtx, td1.CorporationId)
	require.NoError(t, err)
	require.Equal(t, td1, storedTd1)

	storedTd2, err := newK.TrustDeposit.Get(newCtx, td2.CorporationId)
	require.NoError(t, err)
	require.Equal(t, td2, storedTd2)
}

func TestGenesisRoundtripPendingTuDecayRate(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(ctx, k, *types.DefaultGenesis())
	require.Nil(t, trustdeposit.ExportGenesis(ctx, k).PendingTuDecayRate)

	pending := math.LegacyMustNewDecFromStr("0.002")
	gs := *types.DefaultGenesis()
	gs.PendingTuDecayRate = &pending
	k2, ctx2 := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(ctx2, k2, gs)
	got := trustdeposit.ExportGenesis(ctx2, k2)
	require.NotNil(t, got.PendingTuDecayRate)
	require.Equal(t, pending, *got.PendingTuDecayRate)
}

func TestGenesisFirstEpochOrigin(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	genesisTime := time.Unix(1_700_000_123, 500).UTC()
	ctx = ctx.WithBlockTime(genesisTime)

	trustdeposit.InitGenesis(ctx, k, *types.DefaultGenesis())
	got := trustdeposit.ExportGenesis(ctx, k)

	require.Equal(t, types.DefaultTuPegValueGenesis, got.TuPegValue)
	require.Equal(t, time.Unix(1_700_000_123, 0).UTC(), *got.EpochOrigin)
	require.Nil(t, got.LastClosedEpoch)
	require.Nil(t, got.PendingTuDecayRate)
}

func TestGenesisRoundtripTuIndex(t *testing.T) {
	origin := time.Unix(1_700_000_000, 0).UTC()
	closed := uint64(41)
	gs := *types.DefaultGenesis()
	gs.TuPegValue = math.LegacyMustNewDecFromStr("95.9")
	gs.EpochOrigin = &origin
	gs.LastClosedEpoch = &closed

	k, ctx := keepertest.TrustdepositKeeper(t)
	ctx = ctx.WithBlockTime(origin.Add(400 * 24 * time.Hour))
	trustdeposit.InitGenesis(ctx, k, gs)
	got := trustdeposit.ExportGenesis(ctx, k)

	require.Equal(t, gs.TuPegValue, got.TuPegValue)
	require.Equal(t, origin, *got.EpochOrigin)
	require.Equal(t, closed, *got.LastClosedEpoch)
}

func TestGenesisEpochOriginAfterGenesisTime(t *testing.T) {
	origin := time.Unix(1_700_000_000, 0).UTC()
	gs := *types.DefaultGenesis()
	gs.EpochOrigin = &origin

	k, ctx := keepertest.TrustdepositKeeper(t)
	ctx = ctx.WithBlockTime(origin.Add(-time.Second))
	require.Panics(t, func() { trustdeposit.InitGenesis(ctx, k, gs) })
}
