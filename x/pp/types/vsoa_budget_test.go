package types

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// [MOD-DE-MSG-5-2] budget rules as checked at the message level for MSG-1/7/14.
func TestValidateVSOperatorAuthzBudget(t *testing.T) {
	coins := sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
	zero := sdk.Coins{sdk.NewInt64Coin("uvna", 0)}
	period := time.Hour
	bad := -time.Hour

	require.NoError(t, validateVSOperatorAuthzBudget(false, nil, nil, nil))
	require.NoError(t, validateVSOperatorAuthzBudget(true, coins, nil, nil))
	require.NoError(t, validateVSOperatorAuthzBudget(false, nil, coins, &period))

	require.Error(t, validateVSOperatorAuthzBudget(true, nil, nil, nil), "with_feegrant needs a limit")
	require.Error(t, validateVSOperatorAuthzBudget(true, zero, nil, nil), "zero is not a limit")
	require.Error(t, validateVSOperatorAuthzBudget(false, coins, nil, nil), "limit needs with_feegrant")
	require.Error(t, validateVSOperatorAuthzBudget(false, nil, nil, &period), "period needs spend_limit")
	require.Error(t, validateVSOperatorAuthzBudget(false, nil, coins, &bad), "period must be positive")
}

func TestMsgStartParticipantOP_ValidateBasic_Budget(t *testing.T) {
	addr := sdk.AccAddress([]byte("start_op_signer_____")).String()
	vsOp := sdk.AccAddress([]byte("start_op_vsop_______")).String()
	base := func() *MsgStartParticipantOP {
		return &MsgStartParticipantOP{
			Corporation: addr, Operator: addr, Role: ParticipantRole_ISSUER, ValidatorParticipantId: 1,
			Did: "did:example:budget", VsOperator: vsOp,
			VsOperatorAuthzMsgTypes: []detypes.MsgType{detypes.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE},
		}
	}

	ok := base()
	ok.VsOperatorAuthzWithFeegrant = true
	ok.VsOperatorAuthzFeeSpendLimit = sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
	require.NoError(t, ok.ValidateBasic())

	missing := base()
	missing.VsOperatorAuthzWithFeegrant = true
	require.Error(t, missing.ValidateBasic())

	stray := base()
	stray.VsOperatorAuthzFeeSpendLimit = sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
	require.Error(t, stray.ValidateBasic())
}

func TestValidateVSOperatorAuthz_RoleMsgTypes(t *testing.T) {
	vsOp := sdk.AccAddress([]byte("role_vsop___________")).String()
	for _, tc := range []struct {
		role ParticipantRole
		mt   detypes.MsgType
		ok   bool
	}{
		{ParticipantRole_HOLDER, detypes.MsgType_MSG_TYPE_TRIGGER_RESOLVER, true},
		{ParticipantRole_HOLDER, detypes.MsgType_MSG_TYPE_VALIDATE_ONBOARDING, false},
		{ParticipantRole_ISSUER, detypes.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE, true},
		{ParticipantRole_ISSUER, detypes.MsgType_MSG_TYPE_VALIDATE_ONBOARDING, true},
		{ParticipantRole_ISSUER, detypes.MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION, false},
		{ParticipantRole_VERIFIER, detypes.MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION, true},
		{ParticipantRole_VERIFIER, detypes.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE, false},
		{ParticipantRole_ISSUER_GRANTOR, detypes.MsgType_MSG_TYPE_VALIDATE_ONBOARDING, true},
		{ParticipantRole_VERIFIER_GRANTOR, detypes.MsgType_MSG_TYPE_VALIDATE_ONBOARDING, true},
		{ParticipantRole_VERIFIER_GRANTOR, detypes.MsgType_MSG_TYPE_STORE_DIGEST, false},
		{ParticipantRole_ECOSYSTEM, detypes.MsgType_MSG_TYPE_VALIDATE_ONBOARDING, false},
	} {
		err := validateVSOperatorAuthz(tc.role, vsOp, []detypes.MsgType{tc.mt}, true)
		if tc.ok {
			require.NoError(t, err, "%s %s", tc.role, tc.mt)
		} else {
			require.Error(t, err, "%s %s", tc.role, tc.mt)
		}
	}
}
