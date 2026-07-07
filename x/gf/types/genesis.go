package types

import (
	"fmt"

	"github.com/verana-labs/verana-node/util/validation"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Versions:  nil,
		Documents: nil,
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	versionIDs := map[uint64]struct{}{}
	ownerVersion := map[string]struct{}{}
	var maxGFV uint64
	for _, gfv := range gs.Versions {
		// Exactly one of ecosystem_id / corporation_id must be set.
		if (gfv.EcosystemId > 0) == (gfv.CorporationId > 0) {
			return ErrInvalidSubject
		}
		if gfv.Version < 1 {
			return ErrInvalidVersion
		}
		if _, dup := versionIDs[gfv.Id]; dup {
			return ErrInvalidVersion.Wrapf("duplicate gfv id %d", gfv.Id)
		}
		versionIDs[gfv.Id] = struct{}{}
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
	var maxGFD uint64
	for _, gfd := range gs.Documents {
		if _, ok := versionIDs[gfd.GfvId]; !ok {
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
	return nil
}
