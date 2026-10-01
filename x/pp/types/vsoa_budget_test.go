package types

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
			VsOperatorAuthzMsgTypes: []string{MsgCreateOrUpdateParticipantSessionTypeURL},
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
