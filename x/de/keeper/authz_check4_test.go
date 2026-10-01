package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [AUTHZ-CHECK-4] CheckVSOperatorFeeGrant: the with_feegrant gate. The fee cap
// is the aggregate allowance, see TestRecomputeFeeAllowance_Sum.

func TestCheckVSOFeeGrant_NotFound(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	err := f.keeper.CheckVSOperatorFeeGrant(ctx, 10)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

func TestCheckVSOFeeGrant_NotEnabled(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, acc("vsop________________"), types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: false,
	}))

	err := k.CheckVSOperatorFeeGrant(ctx, 10)
	require.ErrorIs(t, err, types.ErrVSOFeegrantNotEnabled)
}

func TestCheckVSOFeeGrant_Enabled(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, acc("vsop________________"), types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true,
		FeeSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 80)),
	}))

	require.NoError(t, k.CheckVSOperatorFeeGrant(ctx, 10))
}
