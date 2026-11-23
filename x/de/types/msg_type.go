package types

import "fmt"

// msgTypeURLs maps each MsgType to the type URLs of the messages that carry it
// on this chain. A value without a row has no message yet.
var msgTypeURLs = map[MsgType][]string{
	MsgType_MSG_TYPE_UPDATE_CORPORATION:                           {"/verana.co.v1.MsgUpdateCorporation"},
	MsgType_MSG_TYPE_CREATE_ECOSYSTEM:                             {"/verana.ec.v1.MsgCreateEcosystem"},
	MsgType_MSG_TYPE_UPDATE_ECOSYSTEM:                             {"/verana.ec.v1.MsgUpdateEcosystem"},
	MsgType_MSG_TYPE_ARCHIVE_ECOSYSTEM:                            {"/verana.ec.v1.MsgArchiveEcosystem"},
	MsgType_MSG_TYPE_ADD_GOVERNANCE_FRAMEWORK_DOCUMENT:            {"/verana.gf.v1.MsgAddGovernanceFrameworkDocument"},
	MsgType_MSG_TYPE_INCREASE_ACTIVE_GOVERNANCE_FRAMEWORK_VERSION: {"/verana.gf.v1.MsgIncreaseActiveGovernanceFrameworkVersion"},
	MsgType_MSG_TYPE_CREATE_CREDENTIAL_SCHEMA:                     {"/verana.cs.v1.MsgCreateCredentialSchema"},
	MsgType_MSG_TYPE_UPDATE_CREDENTIAL_SCHEMA:                     {"/verana.cs.v1.MsgUpdateCredentialSchema"},
	MsgType_MSG_TYPE_ARCHIVE_CREDENTIAL_SCHEMA:                    {"/verana.cs.v1.MsgArchiveCredentialSchema"},
	MsgType_MSG_TYPE_START_ONBOARDING:                             {"/verana.pp.v1.MsgStartParticipantOP"},
	MsgType_MSG_TYPE_RENEW_ONBOARDING:                             {"/verana.pp.v1.MsgRenewParticipantOP"},
	MsgType_MSG_TYPE_VALIDATE_ONBOARDING:                          {"/verana.pp.v1.MsgSetParticipantOPToValidated"},
	MsgType_MSG_TYPE_WITHDRAW_ONBOARDING_REQUEST:                  {"/verana.pp.v1.MsgCancelParticipantOPLastRequest"},
	MsgType_MSG_TYPE_CREATE_ROOT_ACCREDITATION:                    {"/verana.pp.v1.MsgCreateRootParticipant"},
	MsgType_MSG_TYPE_SET_ACCREDITATION_EFFECTIVE_UNTIL:            {"/verana.pp.v1.MsgSetParticipantEffectiveUntil"},
	MsgType_MSG_TYPE_REVOKE_ACCREDITATION:                         {"/verana.pp.v1.MsgRevokeParticipant"},
	MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE:                   {"/verana.pp.v1.MsgCreateOrUpdateParticipantSession"},
	MsgType_MSG_TYPE_SLASH_PARTICIPATION_TRUST_DEPOSIT:            {"/verana.pp.v1.MsgSlashParticipantTrustDeposit"},
	MsgType_MSG_TYPE_SELF_CREATE_ACCREDITATION:                    {"/verana.pp.v1.MsgSelfCreateParticipant"},
	MsgType_MSG_TYPE_TRIGGER_RESOLVER:                             {"/verana.pp.v1.MsgTriggerResolver"},
	MsgType_MSG_TYPE_REPAY_SLASH:                                  {"/verana.td.v1.MsgRepaySlashedTrustDeposit", "/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit"},
	MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION:                 {"/verana.de.v1.MsgGrantOperatorAuthorization"},
	MsgType_MSG_TYPE_REVOKE_DELEGATE_AUTHORIZATION:                {"/verana.de.v1.MsgRevokeOperatorAuthorization"},
	MsgType_MSG_TYPE_STORE_DIGEST:                                 {"/verana.di.v1.MsgStoreDigest"},
	MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION:               {"/verana.pp.v1.MsgCreateOrUpdateParticipantSession"},
}

// MsgTypeSet is the set of values a msg_types list may hold.
type MsgTypeSet map[MsgType]bool

func newMsgTypeSet(mts ...MsgType) MsgTypeSet {
	s := make(MsgTypeSet, len(mts))
	for _, mt := range mts {
		s[mt] = true
	}
	return s
}

func allMsgTypesExcept(except ...MsgType) MsgTypeSet {
	s := make(MsgTypeSet, len(MsgType_name))
	for n := range MsgType_name {
		if n != int32(MsgType_MSG_TYPE_UNSPECIFIED) {
			s[MsgType(n)] = true
		}
	}
	for _, mt := range except {
		delete(s, mt)
	}
	return s
}

var (
	// DelegateAuthorizationMsgTypes: every value except the two reserved to VS
	// accounts and AcceptCorporationGovernanceFramework.
	DelegateAuthorizationMsgTypes = allMsgTypesExcept(
		MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE,
		MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION,
		MsgType_MSG_TYPE_ACCEPT_CORPORATION_GOVERNANCE_FRAMEWORK,
	)
	// FeeGrantMsgTypes: the delegable values plus AcceptCorporationGovernanceFramework.
	FeeGrantMsgTypes          = allMsgTypesExcept()
	VerifiableServiceMsgTypes = newMsgTypeSet(
		MsgType_MSG_TYPE_VALIDATE_ONBOARDING,
		MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE,
		MsgType_MSG_TYPE_TRIGGER_RESOLVER,
		MsgType_MSG_TYPE_STORE_DIGEST,
		MsgType_MSG_TYPE_REVOKE_DIGEST,
		MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION,
	)
	SponsorshipMsgTypes = newMsgTypeSet(
		MsgType_MSG_TYPE_CREATE_ECOSYSTEM,
		MsgType_MSG_TYPE_START_ONBOARDING,
		MsgType_MSG_TYPE_RENEW_ONBOARDING,
		MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE,
		MsgType_MSG_TYPE_REPAY_SLASH,
		MsgType_MSG_TYPE_REGISTER_VERIFIABLE_SERVICE,
		MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION,
	)
)

// URLs returns the type URLs of the messages of mt on this chain.
func (mt MsgType) URLs() []string {
	return msgTypeURLs[mt]
}

// MsgTypeURLs returns the distinct type URLs of mts, in order, for x/feegrant.
func MsgTypeURLs(mts []MsgType) ([]string, error) {
	seen := make(map[string]bool, len(mts))
	urls := make([]string, 0, len(mts))
	for _, mt := range mts {
		if len(mt.URLs()) == 0 {
			return nil, fmt.Errorf("%w: %s has no message on this chain", ErrInvalidMsgType, mt)
		}
		for _, url := range mt.URLs() {
			if !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}
	return urls, nil
}

// ValidateMsgTypes checks that mts is non-empty, holds only values of
// permitted, and that each value has a message on this chain.
func ValidateMsgTypes(mts []MsgType, permitted MsgTypeSet) error {
	if len(mts) == 0 {
		return fmt.Errorf("%w: msg_types must not be empty", ErrInvalidMsgType)
	}
	for _, mt := range mts {
		if !permitted[mt] {
			return fmt.Errorf("%w: %s is not permitted in this list", ErrInvalidMsgType, mt)
		}
		if len(mt.URLs()) == 0 {
			return fmt.Errorf("%w: %s has no message on this chain", ErrInvalidMsgType, mt)
		}
	}
	return nil
}
