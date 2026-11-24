package types_test

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

func TestMsgGrantDelegateAuthorization_ValidateBasicLimits(t *testing.T) {
	corp := sdk.AccAddress([]byte("corp_policy_________")).String()
	op := sdk.AccAddress([]byte("delegate____________")).String()
	limit := sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
	zero := sdk.Coins{sdk.NewInt64Coin("uvna", 0)}
	exp := time.Now().Add(time.Hour)
	day, none := 24*time.Hour, time.Duration(0)
	for _, tc := range []struct {
		name string
		edit func(*types.MsgGrantDelegateAuthorization)
		err  error
	}{
		{"no limits", func(*types.MsgGrantDelegateAuthorization) {}, nil},
		{"authz limit", func(m *types.MsgGrantDelegateAuthorization) { m.AuthzSpendLimit = limit }, nil},
		{"authz limit zero", func(m *types.MsgGrantDelegateAuthorization) { m.AuthzSpendLimit = zero }, types.ErrInvalidSpendLimit},
		{"authz cycle", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimit, m.AuthzSpendLimitPeriod, m.Expiration = limit, &day, &exp
		}, nil},
		{"authz period zero", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimit, m.AuthzSpendLimitPeriod, m.Expiration = limit, &none, &exp
		}, sdkerrors.ErrInvalidRequest},
		{"authz period without expiration", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimit, m.AuthzSpendLimitPeriod = limit, &day
		}, sdkerrors.ErrInvalidRequest},
		{"authz period alone without expiration", func(m *types.MsgGrantDelegateAuthorization) { m.AuthzSpendLimitPeriod = &day }, sdkerrors.ErrInvalidRequest},
		{"feegrant limit", func(m *types.MsgGrantDelegateAuthorization) { m.FeegrantSpendLimit = limit }, nil},
		{"feegrant limit zero", func(m *types.MsgGrantDelegateAuthorization) { m.FeegrantSpendLimit = zero }, types.ErrInvalidSpendLimit},
		{"feegrant cycle", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimit, m.AuthzSpendLimitPeriod, m.Expiration = limit, &day, &exp
			m.FeegrantSpendLimit, m.FeegrantSpendLimitPeriod = limit, &day
		}, nil},
		{"feegrant period zero", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimit, m.AuthzSpendLimitPeriod, m.Expiration = limit, &day, &exp
			m.FeegrantSpendLimit, m.FeegrantSpendLimitPeriod = limit, &none
		}, sdkerrors.ErrInvalidRequest},
		{"feegrant period without expiration", func(m *types.MsgGrantDelegateAuthorization) {
			m.FeegrantSpendLimit, m.FeegrantSpendLimitPeriod = limit, &day
		}, sdkerrors.ErrInvalidRequest},
		{"feegrant period without authz limit", func(m *types.MsgGrantDelegateAuthorization) {
			m.Expiration, m.FeegrantSpendLimit, m.FeegrantSpendLimitPeriod = &exp, limit, &day
		}, sdkerrors.ErrInvalidRequest},
		{"feegrant period without authz period", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimit, m.Expiration = limit, &exp
			m.FeegrantSpendLimit, m.FeegrantSpendLimitPeriod = limit, &day
		}, sdkerrors.ErrInvalidRequest},
		{"feegrant period with authz period only", func(m *types.MsgGrantDelegateAuthorization) {
			m.AuthzSpendLimitPeriod, m.Expiration = &day, &exp
			m.FeegrantSpendLimit, m.FeegrantSpendLimitPeriod = limit, &day
		}, sdkerrors.ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := &types.MsgGrantDelegateAuthorization{
				Corporation: corp, Delegate: corp, Grantee: op,
				MsgTypes: []types.MsgType{types.MsgType_MSG_TYPE_CREATE_ECOSYSTEM},
			}
			tc.edit(msg)
			err := msg.ValidateBasic()
			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.err)
			}
		})
	}
}
