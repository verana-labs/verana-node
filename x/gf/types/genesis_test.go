package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/gf/types"
)

// withVersions returns the default genesis, which holds network version 1
// (id 1, document id 1), plus gfvs and gfds.
func withVersions(gfvs []types.GovernanceFrameworkVersion, gfds []types.GovernanceFrameworkDocument) types.GenesisState {
	gs := *types.DefaultGenesis()
	gs.Versions = append(gs.Versions, gfvs...)
	gs.Documents = append(gs.Documents, gfds...)
	gs.GfvCounter = uint64(len(gs.Versions))
	gs.GfdCounter = uint64(len(gs.Documents))
	return gs
}

func TestGenesisValidate(t *testing.T) {
	t.Run("default genesis is valid", func(t *testing.T) {
		require.NoError(t, types.DefaultGenesis().Validate())
	})

	t.Run("rejects GFV with both ecosystem_id and corporation set", func(t *testing.T) {
		gs := withVersions([]types.GovernanceFrameworkVersion{{Id: 2, EcosystemId: 1, CorporationId: 1, Version: 1}}, nil)
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidSubject)
	})

	t.Run("rejects GFD with dangling gfv_id", func(t *testing.T) {
		gs := withVersions([]types.GovernanceFrameworkVersion{{Id: 2, CorporationId: 1, Version: 1}},
			[]types.GovernanceFrameworkDocument{{Id: 2, GfvId: 999, Language: "en"}})
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidVersion)
	})

	t.Run("rejects GFD with invalid BCP47 language", func(t *testing.T) {
		gs := withVersions([]types.GovernanceFrameworkVersion{{Id: 2, CorporationId: 1, Version: 1}},
			[]types.GovernanceFrameworkDocument{{Id: 2, GfvId: 2, Language: "1bad"}})
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidLanguage)
	})

	t.Run("rejects GFV with version below 1", func(t *testing.T) {
		gs := withVersions([]types.GovernanceFrameworkVersion{{Id: 2, CorporationId: 1, Version: 0}}, nil)
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidVersion)
	})

	t.Run("accepts valid mixed eco + corp GFVs with valid GFDs", func(t *testing.T) {
		gs := withVersions(
			[]types.GovernanceFrameworkVersion{{Id: 2, EcosystemId: 7, Version: 1}, {Id: 3, CorporationId: 9, Version: 1}},
			[]types.GovernanceFrameworkDocument{
				{Id: 2, GfvId: 2, Language: "en", Url: "https://example.com/1", DigestSri: "sha256-abc123"},
				{Id: 3, GfvId: 3, Language: "fr", Url: "https://example.com/2", DigestSri: "sha256-def456"},
			})
		require.NoError(t, gs.Validate())
	})
}

// [NGG]
func TestGenesisValidateNetworkGovernance(t *testing.T) {
	active := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	draft := types.GovernanceFrameworkVersion{Id: 2, Version: 2}
	activeV2 := types.GovernanceFrameworkVersion{Id: 2, Version: 2, ActiveSince: &active}
	tests := []struct {
		name   string
		mutate func(gs *types.GenesisState)
		err    error
	}{
		{"accepts a draft after version 1", func(gs *types.GenesisState) {
			gs.Versions = append(gs.Versions, draft)
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 2}
			gs.GfvCounter = 2
		}, nil},
		{"accepts an exported deadline", func(gs *types.GenesisState) {
			gs.NetworkGovernance.NgfAcceptanceDeadline = &active
		}, nil},
		{"requires version 1 of the network framework", func(gs *types.GenesisState) {
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = nil
		}, types.ErrInvalidSubject},
		{"rejects a listed id that is not a network version", func(gs *types.GenesisState) {
			gs.Versions = append(gs.Versions, types.GovernanceFrameworkVersion{Id: 2, CorporationId: 1, Version: 1})
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 2}
			gs.GfvCounter = 2
		}, types.ErrInvalidSubject},
		{"rejects an unknown listed id", func(gs *types.GenesisState) {
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 9}
		}, types.ErrInvalidSubject},
		{"rejects a network version that is not listed", func(gs *types.GenesisState) {
			gs.Versions = append(gs.Versions, draft)
			gs.GfvCounter = 2
		}, types.ErrInvalidSubject},
		{"rejects a duplicate listed id", func(gs *types.GenesisState) {
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 1}
		}, types.ErrInvalidVersion},
		{"rejects versions out of order", func(gs *types.GenesisState) {
			gs.Versions[0].Version = 2
		}, types.ErrInvalidVersion},
		{"rejects an inactive version before the last", func(gs *types.GenesisState) {
			gs.Versions = append(gs.Versions, draft, types.GovernanceFrameworkVersion{Id: 3, Version: 3, ActiveSince: &active})
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 2, 3}
			gs.GfvCounter = 3
		}, types.ErrInvalidVersion},
		{"accepts an active version after version 1", func(gs *types.GenesisState) {
			gs.Versions = append(gs.Versions, activeV2)
			gs.NetworkGovernance.NetworkGovernanceFrameworkVersions = []uint64{1, 2}
			gs.GfvCounter = 2
		}, nil},
		{"requires a document for version 1", func(gs *types.GenesisState) {
			gs.Documents = nil
		}, types.ErrMissingDefaultLang},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs := types.DefaultGenesis()
			tc.mutate(gs)
			if tc.err == nil {
				require.NoError(t, gs.Validate())
				return
			}
			require.ErrorIs(t, gs.Validate(), tc.err)
		})
	}
}

func TestGenesisValidateEcosystemAcceptances(t *testing.T) {
	accepted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	valid := types.EcosystemAcceptance{EcosystemId: 7, CorporationId: 1, Version: 2, Accepted: accepted}
	tests := []struct {
		name string
		accs []types.EcosystemAcceptance
		err  error
	}{
		{"accepts distinct pairs", []types.EcosystemAcceptance{valid, {EcosystemId: 7, CorporationId: 2, Version: 1, Accepted: accepted}}, nil},
		{"requires the ecosystem", []types.EcosystemAcceptance{{CorporationId: 1, Version: 1, Accepted: accepted}}, types.ErrInvalidSubject},
		{"requires the corporation", []types.EcosystemAcceptance{{EcosystemId: 7, Version: 1, Accepted: accepted}}, types.ErrInvalidSubject},
		{"requires a version", []types.EcosystemAcceptance{{EcosystemId: 7, CorporationId: 1, Accepted: accepted}}, types.ErrInvalidVersion},
		{"requires the acceptance time", []types.EcosystemAcceptance{{EcosystemId: 7, CorporationId: 1, Version: 1}}, types.ErrInvalidVersion},
		{"rejects a duplicate pair", []types.EcosystemAcceptance{valid, valid}, types.ErrInvalidSubject},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs := types.DefaultGenesis()
			gs.EcosystemAcceptances = tc.accs
			if tc.err == nil {
				require.NoError(t, gs.Validate())
				return
			}
			require.ErrorIs(t, gs.Validate(), tc.err)
		})
	}
}
