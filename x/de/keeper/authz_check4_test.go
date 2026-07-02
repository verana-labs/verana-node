package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [AUTHZ-CHECK-4] CheckVSOperatorFeeGrant + ConsumeRecordFeeSpend.

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
	exp := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, acc("vsop________________"), types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true, Expiration: &exp,
	}))

	require.NoError(t, k.CheckVSOperatorFeeGrant(ctx, 10))
}

func TestConsumeRecordFeeSpend(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true,
		FeeSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 80)),
	}))

	// Debit 30 -> remaining 50.
	require.NoError(t, k.ConsumeRecordFeeSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 30))))
	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)
	vsoa, _ := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 50)), vsoa.Records[0].RemainingFeeSpend)

	require.ErrorIs(t, k.ConsumeRecordFeeSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 60))), types.ErrAuthzSpendLimitExceeded)
}
