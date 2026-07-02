package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [AUTHZ-CHECK-3] CheckVSOperatorAuthorizationOnParticipant. Step 5 (spend
// deduct) is covered by TestConsumeRecordSpend.

func TestCheckVSOA_RecordNotFound(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	err := f.keeper.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, acc("vsop________________"), 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

func TestCheckVSOA_WrongCorporation(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	exp := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, Expiration: &exp,
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 999, vsOp, 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

func TestCheckVSOA_WrongOperator(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	exp := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, acc("vsop________________"), types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, Expiration: &exp,
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, acc("vsop2_______________"), 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

func TestCheckVSOA_MsgTypeNotAuthorized(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	exp := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, Expiration: &exp,
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtEcosystem)
	require.ErrorIs(t, err, types.ErrAuthzMsgTypeNotFound)
}

func TestCheckVSOA_Expired(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	past := ctx.BlockTime().Add(-time.Minute)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, Expiration: &past,
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrAuthzExpired)
}

func TestCheckVSOA_Success(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	exp := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, Expiration: &exp,
	}))

	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS))
}

func TestCheckVSOA_PeriodReset(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	past := now.Add(-time.Minute)
	period := time.Hour
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))
	fee := sdk.NewCoins(sdk.NewInt64Coin("uvna", 80))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
		SpendLimit: spend, FeeSpendLimit: fee, WithFeegrant: true,
		Period: &period, Expiration: &past,
	}))

	// Deplete so the reset is observable.
	require.NoError(t, k.ConsumeRecordSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 50))))
	require.NoError(t, k.ConsumeRecordFeeSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 20))))

	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, corpID, vsOp, 10, mtCSPS))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Equal(t, spend, vsoa.Records[0].RemainingSpend)
	require.Equal(t, fee, vsoa.Records[0].RemainingFeeSpend)
	require.True(t, vsoa.Records[0].Expiration.After(now))
}
