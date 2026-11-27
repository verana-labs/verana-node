package keeper_test

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

func TestKeeperGenesis_InitExportRoundTrip(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	t0 := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)

	original := types.GenesisState{
		Params: types.DefaultParams(),
		Versions: []types.GovernanceFrameworkVersion{
			{Id: 1, EcosystemId: 7, Version: 1, Created: t0},
			{Id: 2, EcosystemId: 7, Version: 2, Created: t0.Add(time.Hour)},
			{Id: 3, CorporationId: 9, Version: 1, Created: t0},
		},
		Documents: []types.GovernanceFrameworkDocument{
			{Id: 1, GfvId: 1, Language: "en", Url: "https://example.com/v1-en.html", DigestSri: "sha384-x", Created: t0},
			{Id: 2, GfvId: 1, Language: "fr", Url: "https://example.com/v1-fr.html", DigestSri: "sha384-x", Created: t0},
			{Id: 3, GfvId: 3, Language: "en", Url: "https://example.com/corp-v1-en.html", DigestSri: "sha384-x", Created: t0},
		},
	}

	require.NoError(t, k.InitGenesis(ctx, original))

	exported := k.ExportGenesis(ctx)
	require.NotNil(t, exported)

	require.Equal(t, original.Params, exported.Params)

	// Sort both sides by id for deterministic comparison.
	sortVersions := func(vs []types.GovernanceFrameworkVersion) {
		sort.Slice(vs, func(i, j int) bool { return vs[i].Id < vs[j].Id })
	}
	sortDocs := func(ds []types.GovernanceFrameworkDocument) {
		sort.Slice(ds, func(i, j int) bool { return ds[i].Id < ds[j].Id })
	}
	sortVersions(original.Versions)
	sortVersions(exported.Versions)
	sortDocs(original.Documents)
	sortDocs(exported.Documents)
	require.Equal(t, original.Versions, exported.Versions)
	require.Equal(t, original.Documents, exported.Documents)

	// Counter restoration: next ids must be > max(observed id).
	nextGFV, err := k.GetNextID(ctx, "gfv")
	require.NoError(t, err)
	require.Equal(t, uint64(4), nextGFV, "next gfv id must continue past the highest imported id (3)")

	nextGFD, err := k.GetNextID(ctx, "gfd")
	require.NoError(t, err)
	require.Equal(t, uint64(4), nextGFD, "next gfd id must continue past the highest imported id (3)")
}

// [NGG] the default genesis holds network version 1, created and active at
// genesis time, with its document.
func TestKeeperGenesis_Default(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	genesis := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(genesis)

	require.NoError(t, k.InitGenesis(ctx, *types.DefaultGenesis()))

	exported := k.ExportGenesis(ctx)
	require.NotNil(t, exported)
	require.Equal(t, types.DefaultParams(), exported.Params)
	require.Equal(t, []types.GovernanceFrameworkVersion{{Id: 1, Created: genesis, Version: 1, ActiveSince: &genesis}}, exported.Versions)
	require.Len(t, exported.Documents, 1)
	require.Equal(t, genesis, exported.Documents[0].Created)
	require.Equal(t, types.NetworkGovernance{NetworkGovernanceFrameworkVersions: []uint64{1}}, exported.NetworkGovernance)
	require.NoError(t, exported.Validate())

	v, err := k.NetworkGFVersionInForce(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), v)
}

// [NGG] only version 1 takes the genesis time; a listed draft stays a draft.
func TestKeeperGenesis_NetworkDraftStaysDraft(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	ctx = ctx.WithBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	gs := types.DefaultGenesis()
	gs.Versions = append(gs.Versions, types.GovernanceFrameworkVersion{Id: 2, Version: 2})
	gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 2}
	gs.GfvCounter = 2
	require.NoError(t, gs.Validate())

	require.NoError(t, k.InitGenesis(ctx, *gs))
	v, err := k.NetworkGFVersionInForce(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), v)
	draft, err := k.GFVersion.Get(ctx, 2)
	require.NoError(t, err)
	require.Nil(t, draft.ActiveSince)
}

func TestKeeperGenesis_NetworkAndAcceptancesRoundTrip(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	gs := types.DefaultGenesis()
	gs.Versions[0].Created, gs.Versions[0].ActiveSince = at, &at
	gs.Documents[0].Created = at
	gs.NetworkGovernance.NgfAcceptanceDeadline = &at
	gs.EcosystemAcceptances = []types.EcosystemAcceptance{{EcosystemId: 7, CorporationId: 1, Version: 2, Accepted: at}}

	require.NoError(t, k.InitGenesis(ctx, *gs))
	exported := k.ExportGenesis(ctx)
	require.Equal(t, gs.Versions, exported.Versions)
	require.Equal(t, gs.Documents, exported.Documents)
	require.Equal(t, gs.NetworkGovernance, exported.NetworkGovernance)
	require.Equal(t, gs.EcosystemAcceptances, exported.EcosystemAcceptances)
}
