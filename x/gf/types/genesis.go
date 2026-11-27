package types

import (
	"fmt"

	"github.com/verana-labs/verana-node/util/validation"
)

// Placeholder document of the default network governance framework, to be
// replaced by the genesis of a network.
const (
	DefaultNGFDocLanguage  = "en"
	DefaultNGFDocURL       = "https://example.com/network-governance-framework.pdf"
	DefaultNGFDocDigestSRI = "sha384-OLBgp1GsljhM2TJ+sbHjaiH9txEUvgdDTAzHv2P24donTt6/529l+9Ua0vFImLlb"
)

// DefaultGenesis holds version 1 of the network governance framework, created
// and active at genesis time.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Versions:   []GovernanceFrameworkVersion{{Id: 1, Version: 1}},
		Documents:  []GovernanceFrameworkDocument{{Id: 1, GfvId: 1, Language: DefaultNGFDocLanguage, Url: DefaultNGFDocURL, DigestSri: DefaultNGFDocDigestSRI}},
		GfvCounter: 1,
		GfdCounter: 1,
		NetworkGovernance: NetworkGovernance{
			NetworkGovernanceFrameworkVersions: []uint64{1},
		},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	versions := map[uint64]GovernanceFrameworkVersion{}
	ownerVersion := map[string]struct{}{}
	var maxGFV uint64
	for _, gfv := range gs.Versions {
		if gfv.EcosystemId > 0 && gfv.CorporationId > 0 {
			return ErrInvalidSubject
		}
		if gfv.Version < 1 {
			return ErrInvalidVersion
		}
		if _, dup := versions[gfv.Id]; dup {
			return ErrInvalidVersion.Wrapf("duplicate gfv id %d", gfv.Id)
		}
		versions[gfv.Id] = gfv
		ownerKey := fmt.Sprintf("e%d:c%d:v%d", gfv.EcosystemId, gfv.CorporationId, gfv.Version)
		if _, dup := ownerVersion[ownerKey]; dup {
			return ErrInvalidVersion.Wrapf("duplicate (owner, version) %s", ownerKey)
		}
		ownerVersion[ownerKey] = struct{}{}
		if gfv.Id > maxGFV {
			maxGFV = gfv.Id
		}
	}
	docIDs := map[uint64]struct{}{}
	gfvLang := map[string]struct{}{}
	docCount := map[uint64]int{}
	var maxGFD uint64
	for _, gfd := range gs.Documents {
		if _, ok := versions[gfd.GfvId]; !ok {
			return ErrInvalidVersion
		}
		if !IsValidBCP47(gfd.Language) {
			return ErrInvalidLanguage
		}
		if !IsValidURL(gfd.Url) {
			return ErrInvalidURL.Wrap(gfd.Url)
		}
		if !validation.IsValidDigestSRI(gfd.DigestSri) {
			return ErrInvalidDigestSRI.Wrap(gfd.DigestSri)
		}
		if _, dup := docIDs[gfd.Id]; dup {
			return ErrInvalidVersion.Wrapf("duplicate gfd id %d", gfd.Id)
		}
		docIDs[gfd.Id] = struct{}{}
		langKey := fmt.Sprintf("%d:%s", gfd.GfvId, gfd.Language)
		if _, dup := gfvLang[langKey]; dup {
			return ErrInvalidLanguage.Wrapf("duplicate (gfv_id, language) %s", langKey)
		}
		gfvLang[langKey] = struct{}{}
		docCount[gfd.GfvId]++
		if gfd.Id > maxGFD {
			maxGFD = gfd.Id
		}
	}
	if gs.GfvCounter < maxGFV {
		return ErrInvalidVersion.Wrapf("gfv_counter %d < highest gfv id %d", gs.GfvCounter, maxGFV)
	}
	if gs.GfdCounter < maxGFD {
		return ErrInvalidVersion.Wrapf("gfd_counter %d < highest gfd id %d", gs.GfdCounter, maxGFD)
	}
	if err := gs.NetworkGovernance.validate(versions, docCount); err != nil {
		return err
	}
	return validateEcosystemAcceptances(gs.EcosystemAcceptances)
}

// validate checks [NGG]: the network versions are listed in version order from
// 1, version 1 has a document, and only the first (active from genesis time
// when unset) and the last (a draft) may lack active_since.
func (ng NetworkGovernance) validate(versions map[uint64]GovernanceFrameworkVersion, docCount map[uint64]int) error {
	ids := ng.NetworkGovernanceFrameworkVersions
	if len(ids) == 0 {
		return ErrInvalidSubject.Wrap("network_governance_framework_versions must list version 1 of the network governance framework")
	}
	listed := make(map[uint64]struct{}, len(ids))
	for i, id := range ids {
		gfv, ok := versions[id]
		if !ok || gfv.EcosystemId > 0 || gfv.CorporationId > 0 {
			return ErrInvalidSubject.Wrapf("network_governance_framework_versions[%d]: %d is not a network governance framework version", i, id)
		}
		if _, dup := listed[id]; dup {
			return ErrInvalidVersion.Wrapf("network_governance_framework_versions[%d]: duplicate id %d", i, id)
		}
		listed[id] = struct{}{}
		if gfv.Version != uint32(i+1) {
			return ErrInvalidVersion.Wrapf("network_governance_framework_versions[%d]: version %d, want %d", i, gfv.Version, i+1)
		}
		if gfv.ActiveSince == nil && i > 0 && i < len(ids)-1 {
			return ErrInvalidVersion.Wrapf("network_governance_framework_versions[%d]: version %d is not active", i, gfv.Version)
		}
	}
	for id, gfv := range versions {
		if _, ok := listed[id]; !ok && gfv.EcosystemId == 0 && gfv.CorporationId == 0 {
			return ErrInvalidSubject.Wrapf("network governance framework version %d is not listed", id)
		}
	}
	if docCount[ids[0]] == 0 {
		return ErrMissingDefaultLang.Wrap("version 1 of the network governance framework has no document")
	}
	return nil
}

func validateEcosystemAcceptances(accs []EcosystemAcceptance) error {
	seen := make(map[[2]uint64]struct{}, len(accs))
	for i, acc := range accs {
		if acc.EcosystemId == 0 || acc.CorporationId == 0 {
			return ErrInvalidSubject.Wrapf("ecosystem_acceptances[%d]: ecosystem_id and corporation_id are required", i)
		}
		if acc.Version < 1 {
			return ErrInvalidVersion.Wrapf("ecosystem_acceptances[%d]: version must be >= 1", i)
		}
		if acc.Accepted.IsZero() {
			return ErrInvalidVersion.Wrapf("ecosystem_acceptances[%d]: accepted is required", i)
		}
		key := [2]uint64{acc.EcosystemId, acc.CorporationId}
		if _, dup := seen[key]; dup {
			return ErrInvalidSubject.Wrapf("ecosystem_acceptances[%d]: duplicate (ecosystem_id, corporation_id) (%d, %d)", i, acc.EcosystemId, acc.CorporationId)
		}
		seen[key] = struct{}{}
	}
	return nil
}
