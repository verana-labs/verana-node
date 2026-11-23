package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

func TestMsgTypeNumbers(t *testing.T) {
	want := map[string]int32{
		"MSG_TYPE_UNSPECIFIED":                                  0,
		"MSG_TYPE_UPDATE_CORPORATION":                           1,
		"MSG_TYPE_CREATE_ECOSYSTEM":                             2,
		"MSG_TYPE_UPDATE_ECOSYSTEM":                             3,
		"MSG_TYPE_ARCHIVE_ECOSYSTEM":                            4,
		"MSG_TYPE_ADD_GOVERNANCE_FRAMEWORK_DOCUMENT":            5,
		"MSG_TYPE_INCREASE_ACTIVE_GOVERNANCE_FRAMEWORK_VERSION": 6,
		"MSG_TYPE_CREATE_CREDENTIAL_SCHEMA":                     7,
		"MSG_TYPE_UPDATE_CREDENTIAL_SCHEMA":                     8,
		"MSG_TYPE_ARCHIVE_CREDENTIAL_SCHEMA":                    9,
		"MSG_TYPE_START_ONBOARDING":                             10,
		"MSG_TYPE_RENEW_ONBOARDING":                             11,
		"MSG_TYPE_VALIDATE_ONBOARDING":                          12,
		"MSG_TYPE_WITHDRAW_ONBOARDING_REQUEST":                  13,
		"MSG_TYPE_CREATE_ROOT_ACCREDITATION":                    14,
		"MSG_TYPE_SET_ACCREDITATION_EFFECTIVE_UNTIL":            15,
		"MSG_TYPE_REVOKE_ACCREDITATION":                         16,
		"MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE":                   17,
		"MSG_TYPE_SLASH_PARTICIPATION_TRUST_DEPOSIT":            18,
		"MSG_TYPE_SELF_CREATE_ACCREDITATION":                    19,
		"MSG_TYPE_TRIGGER_RESOLVER":                             20,
		"MSG_TYPE_REPAY_SLASH":                                  21,
		"MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION":                 22,
		"MSG_TYPE_REVOKE_DELEGATE_AUTHORIZATION":                23,
		"MSG_TYPE_STORE_DIGEST":                                 24,
		"MSG_TYPE_REVOKE_DIGEST":                                25,
		"MSG_TYPE_REGISTER_VERIFIABLE_SERVICE":                  26,
		"MSG_TYPE_UPDATE_VERIFIABLE_SERVICE":                    27,
		"MSG_TYPE_SET_PARTICIPATION_EXCLUSION":                  28,
		"MSG_TYPE_REVOKE_SPONSORSHIP":                           29,
		"MSG_TYPE_ACCEPT_GOVERNANCE_FRAMEWORK_VERSION":          30,
		"MSG_TYPE_ACCEPT_CORPORATION_GOVERNANCE_FRAMEWORK":      31,
		"MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION":               32,
	}
	require.Equal(t, want, types.MsgType_value)
}

func msgTypes(nums ...int32) []types.MsgType {
	out := make([]types.MsgType, len(nums))
	for i, n := range nums {
		out[i] = types.MsgType(n)
	}
	return out
}

func msgTypeRange(from, to int32, except ...int32) []types.MsgType {
	skip := map[int32]bool{}
	for _, n := range except {
		skip[n] = true
	}
	var out []types.MsgType
	for n := from; n <= to; n++ {
		if !skip[n] {
			out = append(out, types.MsgType(n))
		}
	}
	return out
}

func setValues(s types.MsgTypeSet) []types.MsgType {
	var out []types.MsgType
	for n := int32(0); n <= 64; n++ {
		if s[types.MsgType(n)] {
			out = append(out, types.MsgType(n))
		}
	}
	return out
}

func TestMsgTypeSets(t *testing.T) {
	require.Equal(t, msgTypeRange(1, 32, 17, 31, 32), setValues(types.DelegateAuthorizationMsgTypes))
	require.Equal(t, msgTypeRange(1, 32), setValues(types.FeeGrantMsgTypes))
	require.Equal(t, msgTypes(12, 17, 20, 24, 25, 32), setValues(types.VerifiableServiceMsgTypes))
	require.Equal(t, msgTypes(2, 10, 11, 17, 21, 26, 32), setValues(types.SponsorshipMsgTypes))
}

func TestMsgTypeURL(t *testing.T) {
	want := map[types.MsgType][]string{
		types.MsgType_MSG_TYPE_UPDATE_CORPORATION:                           {"/verana.co.v1.MsgUpdateCorporation"},
		types.MsgType_MSG_TYPE_CREATE_ECOSYSTEM:                             {"/verana.ec.v1.MsgCreateEcosystem"},
		types.MsgType_MSG_TYPE_UPDATE_ECOSYSTEM:                             {"/verana.ec.v1.MsgUpdateEcosystem"},
		types.MsgType_MSG_TYPE_ARCHIVE_ECOSYSTEM:                            {"/verana.ec.v1.MsgArchiveEcosystem"},
		types.MsgType_MSG_TYPE_ADD_GOVERNANCE_FRAMEWORK_DOCUMENT:            {"/verana.gf.v1.MsgAddGovernanceFrameworkDocument"},
		types.MsgType_MSG_TYPE_INCREASE_ACTIVE_GOVERNANCE_FRAMEWORK_VERSION: {"/verana.gf.v1.MsgIncreaseActiveGovernanceFrameworkVersion"},
		types.MsgType_MSG_TYPE_CREATE_CREDENTIAL_SCHEMA:                     {"/verana.cs.v1.MsgCreateCredentialSchema"},
		types.MsgType_MSG_TYPE_UPDATE_CREDENTIAL_SCHEMA:                     {"/verana.cs.v1.MsgUpdateCredentialSchema"},
		types.MsgType_MSG_TYPE_ARCHIVE_CREDENTIAL_SCHEMA:                    {"/verana.cs.v1.MsgArchiveCredentialSchema"},
		types.MsgType_MSG_TYPE_START_ONBOARDING:                             {"/verana.pp.v1.MsgStartParticipantOP"},
		types.MsgType_MSG_TYPE_RENEW_ONBOARDING:                             {"/verana.pp.v1.MsgRenewParticipantOP"},
		types.MsgType_MSG_TYPE_VALIDATE_ONBOARDING:                          {"/verana.pp.v1.MsgSetParticipantOPToValidated"},
		types.MsgType_MSG_TYPE_WITHDRAW_ONBOARDING_REQUEST:                  {"/verana.pp.v1.MsgCancelParticipantOPLastRequest"},
		types.MsgType_MSG_TYPE_CREATE_ROOT_ACCREDITATION:                    {"/verana.pp.v1.MsgCreateRootParticipant"},
		types.MsgType_MSG_TYPE_SET_ACCREDITATION_EFFECTIVE_UNTIL:            {"/verana.pp.v1.MsgSetParticipantEffectiveUntil"},
		types.MsgType_MSG_TYPE_REVOKE_ACCREDITATION:                         {"/verana.pp.v1.MsgRevokeParticipant"},
		types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE:                   {"/verana.pp.v1.MsgCreateOrUpdateParticipantSession"},
		types.MsgType_MSG_TYPE_SLASH_PARTICIPATION_TRUST_DEPOSIT:            {"/verana.pp.v1.MsgSlashParticipantTrustDeposit"},
		types.MsgType_MSG_TYPE_SELF_CREATE_ACCREDITATION:                    {"/verana.pp.v1.MsgSelfCreateParticipant"},
		types.MsgType_MSG_TYPE_TRIGGER_RESOLVER:                             {"/verana.pp.v1.MsgTriggerResolver"},
		types.MsgType_MSG_TYPE_REPAY_SLASH:                                  {"/verana.td.v1.MsgRepaySlashedTrustDeposit", "/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit"},
		types.MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION:                 {"/verana.de.v1.MsgGrantOperatorAuthorization"},
		types.MsgType_MSG_TYPE_REVOKE_DELEGATE_AUTHORIZATION:                {"/verana.de.v1.MsgRevokeOperatorAuthorization"},
		types.MsgType_MSG_TYPE_STORE_DIGEST:                                 {"/verana.di.v1.MsgStoreDigest"},
		types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION:               {"/verana.pp.v1.MsgCreateOrUpdateParticipantSession"},
	}
	for n := range types.MsgType_name {
		mt := types.MsgType(n)
		require.Equal(t, want[mt], mt.URLs(), mt.String())
	}
	require.Empty(t, types.MsgType(33).URLs())
}

func TestMsgTypeURLs(t *testing.T) {
	urls, err := types.MsgTypeURLs(msgTypes(17, 1, 32, 21, 1))
	require.NoError(t, err)
	require.Equal(t, []string{
		"/verana.pp.v1.MsgCreateOrUpdateParticipantSession",
		"/verana.co.v1.MsgUpdateCorporation",
		"/verana.td.v1.MsgRepaySlashedTrustDeposit",
		"/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit",
	}, urls)

	_, err = types.MsgTypeURLs(msgTypes(1, 25))
	require.ErrorIs(t, err, types.ErrInvalidMsgType)
}

func TestValidateMsgTypes(t *testing.T) {
	cases := []struct {
		name string
		list []types.MsgType
		set  types.MsgTypeSet
		ok   bool
	}{
		{"delegate ok", msgTypes(1, 10, 24), types.DelegateAuthorizationMsgTypes, true},
		{"empty", nil, types.DelegateAuthorizationMsgTypes, false},
		{"unspecified", msgTypes(0), types.FeeGrantMsgTypes, false},
		{"unknown value", msgTypes(33), types.FeeGrantMsgTypes, false},
		{"issuance not delegable", msgTypes(1, 17), types.DelegateAuthorizationMsgTypes, false},
		{"presentation not delegable", msgTypes(32), types.DelegateAuthorizationMsgTypes, false},
		{"cgf acceptance not delegable", msgTypes(31), types.DelegateAuthorizationMsgTypes, false},
		{"fee grant session", msgTypes(17, 32), types.FeeGrantMsgTypes, true},
		{"vs ok", msgTypes(12, 17, 20, 24, 32), types.VerifiableServiceMsgTypes, true},
		{"vs other value", msgTypes(10), types.VerifiableServiceMsgTypes, false},
		{"sponsorship ok", msgTypes(2, 10, 11, 17, 21, 32), types.SponsorshipMsgTypes, true},
		{"sponsorship other value", msgTypes(12), types.SponsorshipMsgTypes, false},
		{"permitted but no message yet", msgTypes(25), types.VerifiableServiceMsgTypes, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateMsgTypes(tc.list, tc.set)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, types.ErrInvalidMsgType)
			}
		})
	}
}
