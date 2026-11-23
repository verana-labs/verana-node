package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/keeper"
	"github.com/verana-labs/verana-node/x/de/types"
)

// [AUTHZ-CHECK-1] step 2: a period-bearing delegate authorization auto-renews on
// the first check after expiry instead of returning ErrAuthzExpired (#324).
func TestDelegateAuthzPeriodRenewal(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()

	period := time.Hour
	exp := now.Add(time.Hour)
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 100))

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation:           corporation,
		Delegate:              corporation,
		Grantee:               grantee,
		MsgTypes:              []types.MsgType{mtEcosystem},
		AuthzSpendLimit:       spend,
		AuthzSpendLimitPeriod: &period,
		Expiration:            &exp,
	})
	require.NoError(t, err)

	list, err := qs.ListDelegateAuthorizations(ctx, &types.QueryListDelegateAuthorizationsRequest{Delegate: grantee})
	require.NoError(t, err)
	require.Len(t, list.DelegateAuthorizations, 1)
	id := list.DelegateAuthorizations[0].Id
	// Seeded at grant time.
	require.Equal(t, spend, list.DelegateAuthorizations[0].RemainingSpend)

	// Check well past expiry: must succeed (auto-renew), not abort.
	checkAt := now.Add(90 * time.Minute)
	require.NoError(t, f.keeper.CheckDelegateAuthorization(ctx, corporation, grantee, mtEcosystem, checkAt))

	got, err := qs.GetDelegateAuthorization(ctx, &types.QueryGetDelegateAuthorizationRequest{Id: id})
	require.NoError(t, err)
	require.True(t, got.DelegateAuthorization.Expiration.After(checkAt), "expiration rolled forward")
	require.Equal(t, spend, got.DelegateAuthorization.RemainingSpend, "remaining reset on renewal")
}

// A non-period authorization still aborts once expired.
func TestDelegateAuthzExpiresWithoutPeriod(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()
	exp := now.Add(time.Hour)

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem}, Expiration: &exp,
	})
	require.NoError(t, err)

	err = f.keeper.CheckDelegateAuthorization(ctx, corporation, grantee, mtEcosystem, now.Add(2*time.Hour))
	require.ErrorIs(t, err, types.ErrAuthzExpired)
}

// The current message type must be in msg_types; no list holds UNSPECIFIED.
func TestDelegateAuthzMsgTypeMembership(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)

	require.NoError(t, f.keeper.CheckDelegateAuthorization(ctx, corporation, grantee, mtEcosystem, now))
	require.ErrorIs(t, f.keeper.CheckDelegateAuthorization(ctx, corporation, grantee, mtSchema, now), types.ErrAuthzMsgTypeNotFound)
	require.ErrorIs(t, f.keeper.CheckDelegateAuthorization(ctx, corporation, grantee, types.MsgType_MSG_TYPE_UNSPECIFIED, now), types.ErrAuthzMsgTypeNotFound)
}

// [MOD-DE-MSG-1] FeeGrant seeds remaining_spend at grant.
func TestFeeGrantSeedsRemaining(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")
	corpID := uint64(7)
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 500))

	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, []types.MsgType{mtEcosystem}, nil, spend, nil))
	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, spend, fg.RemainingSpend)
}

// [MOD-DE-MSG-5] VSOA record seeds remaining_spend at creation.
func TestVSOARecordSeedsRemaining(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, SpendLimit: spend,
	}))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Len(t, vsoa.Records, 1)
	require.Equal(t, spend, vsoa.Records[0].RemainingSpend)
}

// [AUTHZ-CHECK-1] step 3: ConsumeDelegateSpend debits remaining_spend and rejects
// over-limit spends and an empty delegate.
func TestConsumeDelegateSpend(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
		AuthzSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 100)),
	})
	require.NoError(t, err)
	list, err := qs.ListDelegateAuthorizations(ctx, &types.QueryListDelegateAuthorizationsRequest{Delegate: grantee})
	require.NoError(t, err)
	id := list.DelegateAuthorizations[0].Id

	// Debit 60 -> remaining 40.
	require.NoError(t, f.keeper.ConsumeDelegateSpend(ctx, corporation, grantee, mtEcosystem, now, sdk.NewCoins(sdk.NewInt64Coin("uvna", 60))))
	got, err := qs.GetDelegateAuthorization(ctx, &types.QueryGetDelegateAuthorizationRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 40)), got.DelegateAuthorization.RemainingSpend)

	// Debit another 60 -> exceeds remaining 40.
	require.ErrorIs(t,
		f.keeper.ConsumeDelegateSpend(ctx, corporation, grantee, mtEcosystem, now, sdk.NewCoins(sdk.NewInt64Coin("uvna", 60))),
		types.ErrAuthzSpendLimitExceeded)

	// Empty delegate is not a path.
	require.ErrorIs(t,
		f.keeper.ConsumeDelegateSpend(ctx, corporation, "", mtEcosystem, now, sdk.NewCoins(sdk.NewInt64Coin("uvna", 999))),
		types.ErrAuthzNotFound)
}

// [AUTHZ-CHECK-1] an empty delegate is rejected, not skipped.
func TestCheckDelegateAuthorization_EmptyDelegate(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	corporation := acc("corp________________")

	require.ErrorIs(t,
		f.keeper.CheckDelegateAuthorization(ctx, corporation, "", mtEcosystem, ctx.BlockTime()),
		types.ErrAuthzNotFound)
}

// Only the messages that offer a group-proposal path accept delegate ==
// corporation; the shared check grants no such bypass.
func TestCheckDelegateAuthorization_CorporationAsDelegate(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	corporation := acc("corp________________")

	require.ErrorIs(t,
		f.keeper.CheckDelegateAuthorization(ctx, corporation, corporation, mtEcosystem, ctx.BlockTime()),
		types.ErrAuthzNotFound)
	require.ErrorIs(t,
		f.keeper.ConsumeDelegateSpend(ctx, corporation, corporation, mtEcosystem, ctx.BlockTime(), sdk.NewCoins(sdk.NewInt64Coin("uvna", 1))),
		types.ErrAuthzNotFound)
}

// No spend_limit configured -> ConsumeDelegateSpend is a no-op (unlimited).
func TestConsumeDelegateSpend_NoLimit(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()
	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)
	require.NoError(t, f.keeper.ConsumeDelegateSpend(ctx, corporation, grantee, mtEcosystem, now, sdk.NewCoins(sdk.NewInt64Coin("uvna", 999))))
}

// [AUTHZ-CHECK-3] step 6: ConsumeRecordSpend debits the record's remaining_spend.
func TestConsumeRecordSpend(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance},
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 200)),
	}))

	// Debit 120 -> remaining 80.
	require.NoError(t, k.ConsumeRecordSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 120))))
	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)
	vsoa, _ := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 80)), vsoa.Records[0].RemainingSpend)

	// Over-limit -> rejected.
	require.ErrorIs(t, k.ConsumeRecordSpend(ctx, corpID, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 120))), types.ErrAuthzSpendLimitExceeded)

	// Wrong corp -> not found.
	require.Error(t, k.ConsumeRecordSpend(ctx, 999, vsOp, 10, sdk.NewCoins(sdk.NewInt64Coin("uvna", 1))))
}
