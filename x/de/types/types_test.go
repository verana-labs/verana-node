package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

func TestIsOwnPolicyAddress(t *testing.T) {
	policy := sdk.AccAddress([]byte("corp_policy_________")).String()
	other := sdk.AccAddress([]byte("delegate____________")).String()

	require.True(t, types.IsOwnPolicyAddress(policy, policy))
	require.True(t, types.IsOwnPolicyAddress(policy, strings.ToUpper(policy)))
	require.False(t, types.IsOwnPolicyAddress(policy, other))
	require.False(t, types.IsOwnPolicyAddress(policy, ""))
	require.False(t, types.IsOwnPolicyAddress(policy, "not-an-address"))
	require.False(t, types.IsOwnPolicyAddress("", ""))
}

func TestMsgGrantDelegateAuthorization_ValidateBasicMsgTypes(t *testing.T) {
	corp := sdk.AccAddress([]byte("corp_policy_________")).String()
	op := sdk.AccAddress([]byte("delegate____________")).String()
	for _, tc := range []struct {
		name string
		mts  []types.MsgType
		ok   bool
	}{
		{"delegable", []types.MsgType{types.MsgType_MSG_TYPE_CREATE_ECOSYSTEM, types.MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION}, true},
		{"empty", nil, false},
		{"issuance", []types.MsgType{types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE}, false},
		{"presentation", []types.MsgType{types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION}, false},
		{"cgf acceptance", []types.MsgType{types.MsgType_MSG_TYPE_ACCEPT_CORPORATION_GOVERNANCE_FRAMEWORK}, false},
		{"unspecified", []types.MsgType{types.MsgType_MSG_TYPE_UNSPECIFIED}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := &types.MsgGrantDelegateAuthorization{Corporation: corp, Delegate: corp, Grantee: op, MsgTypes: tc.mts}
			err := msg.ValidateBasic()
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, types.ErrInvalidMsgType)
			}
		})
	}
}
