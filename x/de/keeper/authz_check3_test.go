package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [AUTHZ-CHECK-3] CheckVSOperatorAuthorizationOnParticipant. Step 6 (spend
// deduct) is covered by TestConsumeRecordSpend.

func TestCheckVSOA_RecordNotFound(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	err := f.keeper.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, acc("vsop________________"), 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

// Step 1: the entry must be an active participant, before any record lookup.
func TestCheckVSOA_ParticipantNotActive(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
	}))

	f.participants.views[10] = types.ParticipantView{} // pending: never validated
	require.ErrorIs(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS), types.ErrParticipantNotActive)

	f.participants.views[10] = types.ParticipantView{Future: true}
	require.ErrorIs(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS), types.ErrParticipantNotActive)

	delete(f.participants.views, 10)
	f.participants.missing[10] = true
	require.ErrorIs(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS), types.ErrParticipantNotActive)

	delete(f.participants.missing, 10)
	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS))
}

// A pending record with a period is not renewed into life: step 1 aborts first.
func TestCheckVSOA_PendingRecordWithPeriodStaysDisabled(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	period := time.Hour
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, SpendLimit: spend, Period: &period,
	}))
	f.participants.views[10] = types.ParticipantView{}

	later := ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Hour))
	require.ErrorIs(t, k.CheckVSOperatorAuthorizationOnParticipant(later, 1, vsOp, 10, mtCSPS), types.ErrParticipantNotActive)

	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)
	vsoa, _ := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Nil(t, vsoa.Records[0].Expiration, "no cycle started")
}

func TestCheckVSOA_WrongCorporation(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 999, vsOp, 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

func TestCheckVSOA_WrongOperator(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, acc("vsop________________"), types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, acc("vsop2_______________"), 10, mtCSPS)
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzNotFound)
}

func TestCheckVSOA_MsgTypeNotAuthorized(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
	}))

	err := k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtEcosystem)
	require.ErrorIs(t, err, types.ErrAuthzMsgTypeNotFound)
}

func TestCheckVSOA_Success(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
	}))

	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, 1, vsOp, 10, mtCSPS))
}

// Period with a cycle not yet started: no reset.
func TestCheckVSOA_NilExpirationWithPeriod(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	period := time.Hour
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
		SpendLimit: spend, Period: &period,
	}))

	require.NoError(t, k.ConsumeRecordSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 50))))

	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, corpID, vsOp, 10, mtCSPS))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 150)), vsoa.Records[0].RemainingSpend)
	require.Nil(t, vsoa.Records[0].Expiration)
}

// Step 5: a due cycle resets remaining_spend and advances expiration; it never aborts.
func TestCheckVSOA_PeriodReset(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	past := now.Add(-time.Minute)
	period := time.Hour
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
		SpendLimit: spend, Period: &period, Expiration: &past,
	}))

	// Deplete so the reset is observable.
	require.NoError(t, k.ConsumeRecordSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 50))))

	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, corpID, vsOp, 10, mtCSPS))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Equal(t, spend, vsoa.Records[0].RemainingSpend)
	require.True(t, vsoa.Records[0].Expiration.Equal(now.Add(period)), "advanced from now, not from the old boundary")
}

// Step 5: an open cycle leaves the balance alone.
func TestCheckVSOA_CycleNotDue(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	future := ctx.BlockTime().Add(time.Hour)
	period := time.Hour
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS},
		SpendLimit: spend, Period: &period, Expiration: &future,
	}))
	require.NoError(t, k.ConsumeRecordSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 50))))

	require.NoError(t, k.CheckVSOperatorAuthorizationOnParticipant(ctx, corpID, vsOp, 10, mtCSPS))

	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)
	vsoa, _ := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 150)), vsoa.Records[0].RemainingSpend)
	require.True(t, vsoa.Records[0].Expiration.Equal(future))
}
