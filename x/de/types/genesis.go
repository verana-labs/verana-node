package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                   DefaultParams(),
		DelegateAuthorizations:   []DelegateAuthorization{},
		FeeGrants:                []FeeGrant{},
		VsOperatorAuthorizations: []VSOperatorAuthorization{},
		WindowEndQueue:           []WindowEndQueueEntry{},
		Sponsorships:             []Sponsorship{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// DelegateAuthorization: (corporation_id, delegate) key unique.
	daKeys := make(map[string]bool)
	for i, da := range gs.DelegateAuthorizations {
		if da.CorporationId == 0 {
			return fmt.Errorf("delegate_authorizations[%d]: corporation_id cannot be 0", i)
		}
		delegate, err := sdk.AccAddressFromBech32(da.Delegate)
		if err != nil {
			return fmt.Errorf("delegate_authorizations[%d]: invalid delegate address: %w", i, err)
		}
		if err := ValidateMsgTypes(da.MsgTypes, DelegateAuthorizationMsgTypes); err != nil {
			return fmt.Errorf("delegate_authorizations[%d]: %w", i, err)
		}
		if !da.SpendLimit.IsValid() || !da.RemainingSpend.IsValid() || !da.SpendLimit.IsAllGTE(da.RemainingSpend) {
			return fmt.Errorf("delegate_authorizations[%d]: spend_limit and remaining_spend must be valid, remaining within the limit", i)
		}
		if da.Period != nil && (*da.Period <= 0 || len(da.SpendLimit) == 0 || da.Expiration == nil) {
			return fmt.Errorf("delegate_authorizations[%d]: period must be positive and requires spend_limit and expiration", i)
		}
		key := fmt.Sprintf("%d/%s", da.CorporationId, delegate)
		if daKeys[key] {
			return fmt.Errorf("delegate_authorizations[%d]: duplicate (corporation_id, delegate) %s", i, key)
		}
		daKeys[key] = true
	}

	// FeeGrant: composite key (grantor_corporation_id, grantee) unique.
	fgKeys := make(map[string]bool)
	for i, fg := range gs.FeeGrants {
		if fg.GrantorCorporationId == 0 {
			return fmt.Errorf("fee_grants[%d]: grantor_corporation_id cannot be 0", i)
		}
		if _, err := sdk.AccAddressFromBech32(fg.Grantee); err != nil {
			return fmt.Errorf("fee_grants[%d]: invalid grantee address: %w", i, err)
		}
		if err := ValidateMsgTypes(fg.MsgTypes, FeeGrantMsgTypes); err != nil {
			return fmt.Errorf("fee_grants[%d]: %w", i, err)
		}
		key := fmt.Sprintf("%d/%s", fg.GrantorCorporationId, fg.Grantee)
		if fgKeys[key] {
			return fmt.Errorf("fee_grants[%d]: duplicate (grantor_corporation_id, grantee) %s", i, key)
		}
		fgKeys[key] = true
	}

	// VSOperatorAuthorization: id (key) non-zero + unique; (corporation_id,
	// vs_operator) unique; the entry exists iff it has at least one record;
	// each record's participant_id is globally unique across all VSOAs.
	vsoaIDs := make(map[uint64]bool)
	vsoaCorpOp := make(map[string]bool)
	participantIDs := make(map[uint64]bool)
	for i, vsoa := range gs.VsOperatorAuthorizations {
		if vsoa.Id == 0 {
			return fmt.Errorf("vs_operator_authorizations[%d]: id cannot be 0", i)
		}
		if vsoaIDs[vsoa.Id] {
			return fmt.Errorf("vs_operator_authorizations[%d]: duplicate id %d", i, vsoa.Id)
		}
		vsoaIDs[vsoa.Id] = true
		if vsoa.CorporationId == 0 {
			return fmt.Errorf("vs_operator_authorizations[%d]: corporation_id cannot be 0", i)
		}
		if _, err := sdk.AccAddressFromBech32(vsoa.VsOperator); err != nil {
			return fmt.Errorf("vs_operator_authorizations[%d]: invalid vs_operator address: %w", i, err)
		}
		idx := fmt.Sprintf("%d/%s", vsoa.CorporationId, vsoa.VsOperator)
		if vsoaCorpOp[idx] {
			return fmt.Errorf("vs_operator_authorizations[%d]: duplicate (corporation_id, vs_operator) %s", i, idx)
		}
		vsoaCorpOp[idx] = true
		if len(vsoa.Records) == 0 {
			return fmt.Errorf("vs_operator_authorizations[%d]: must have at least one record", i)
		}
		for j, rec := range vsoa.Records {
			if rec.ParticipantId == 0 {
				return fmt.Errorf("vs_operator_authorizations[%d].records[%d]: participant_id cannot be 0", i, j)
			}
			if participantIDs[rec.ParticipantId] {
				return fmt.Errorf("vs_operator_authorizations[%d].records[%d]: duplicate participant_id %d (must be globally unique)", i, j, rec.ParticipantId)
			}
			participantIDs[rec.ParticipantId] = true
			if err := ValidateMsgTypes(rec.MsgTypes, VerifiableServiceMsgTypes); err != nil {
				return fmt.Errorf("vs_operator_authorizations[%d].records[%d]: %w", i, j, err)
			}
		}
	}

	for i, e := range gs.WindowEndQueue {
		if e.ParticipantId == 0 {
			return fmt.Errorf("window_end_queue[%d]: participant_id cannot be 0", i)
		}
	}

	spKeys := make(map[[2]uint64]bool)
	for i, s := range gs.Sponsorships {
		if s.SponsorCorporationId == 0 || s.SponsoredCorporationId == 0 {
			return fmt.Errorf("sponsorships[%d]: corporation ids cannot be 0", i)
		}
		if s.SponsorCorporationId == s.SponsoredCorporationId {
			return fmt.Errorf("sponsorships[%d]: a corporation cannot sponsor itself", i)
		}
		key := [2]uint64{s.SponsorCorporationId, s.SponsoredCorporationId}
		if spKeys[key] {
			return fmt.Errorf("sponsorships[%d]: duplicate (sponsor_corporation_id, sponsored_corporation_id) %v", i, key)
		}
		spKeys[key] = true
		if len(s.MsgTypes) > 0 {
			if err := ValidateMsgTypes(s.MsgTypes, SponsorshipMsgTypes); err != nil {
				return fmt.Errorf("sponsorships[%d]: %w", i, err)
			}
		}
		if s.Period != nil && s.Expiration == nil {
			return fmt.Errorf("sponsorships[%d]: period requires expiration", i)
		}
	}

	return nil
}
