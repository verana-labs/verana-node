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

func setupMsgServer(t *testing.T) (*fixture, types.MsgServer, sdk.Context) {
	t.Helper()
	f := initFixture(t)
	ctx := sdk.UnwrapSDKContext(f.ctx)
	return f, keeper.NewMsgServerImpl(f.keeper), ctx
}

func acc(s string) string { return sdk.AccAddress([]byte(s)).String() }

// vpr msg types used in records / fee grants (must be VPR delegable).
const (
	mtEcosystem = "/verana.ec.v1.MsgCreateEcosystem"
	mtSchema    = "/verana.cs.v1.MsgCreateCredentialSchema"
	mtValidated = "/verana.pp.v1.MsgSetParticipantOPToValidated"
	mtCSPS      = "/verana.pp.v1.MsgCreateOrUpdateParticipantSession"
)

func TestMsgServer(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-1/2] Fee allowance (internal keeper methods, by corporation_id)
// ---------------------------------------------------------------------------

func TestGrantAndRevokeFeeAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")
	corpID := uint64(7)
	msgTypes := []string{mtEcosystem}

	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, msgTypes, nil, nil, nil))
	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, corpID, fg.GrantorCorporationId)
	require.Equal(t, grantee, fg.Grantee)
	require.Equal(t, msgTypes, fg.MsgTypes)

	// Update in place with new msg types.
	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, []string{mtSchema}, nil, nil, nil))
	fg, err = k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, []string{mtSchema}, fg.MsgTypes)

	require.NoError(t, k.RevokeFeeAllowance(ctx, corpID, grantee))
	has, err := k.FeeGrants.Has(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.False(t, has)
}

func TestFeeAllowance_Validation(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")

	// non-delegable msg type
	require.Error(t, k.GrantFeeAllowance(ctx, 1, grantee, []string{"/verana.de.v1.MsgUpdateParams"}, nil, nil, nil))
	// empty msg types
	require.Error(t, k.GrantFeeAllowance(ctx, 1, grantee, nil, nil, nil, nil))

	require.ErrorContains(t, k.RevokeFeeAllowance(ctx, 0, grantee), "grantor_corporation_id")
	require.ErrorContains(t, k.RevokeFeeAllowance(ctx, 1, ""), "grantee")
	// no-op when absent
	require.NoError(t, k.RevokeFeeAllowance(ctx, 1, acc("absent______________")))
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-3/4] Grant / Revoke Operator Authorization (msg server)
// ---------------------------------------------------------------------------

func TestGrantOperatorAuthorization_NewAndUpdate(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem},
	})
	require.NoError(t, err)

	list, err := qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Len(t, list.OperatorAuthorizations, 1)
	firstID := list.OperatorAuthorizations[0].Id
	require.NotZero(t, firstID)
	require.Equal(t, grantee, list.OperatorAuthorizations[0].Operator)

	// Re-grant is an in-place update that preserves the id.
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtSchema},
	})
	require.NoError(t, err)
	list, err = qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Len(t, list.OperatorAuthorizations, 1)
	require.Equal(t, firstID, list.OperatorAuthorizations[0].Id)
	require.Equal(t, []string{mtSchema}, list.OperatorAuthorizations[0].MsgTypes)

	// [MOD-DE-QRY-3] get by id.
	got, err := qs.GetOperatorAuthorization(ctx, &types.QueryGetOperatorAuthorizationRequest{Id: firstID})
	require.NoError(t, err)
	require.Equal(t, grantee, got.OperatorAuthorization.Operator)
}

func TestGrantOperatorAuthorization_MutualExclusivity(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	// Resolve the signing corporation to its co.id (same path MSG-3 uses).
	co, err := f.corpKeeper.ResolveCorporationByPolicyAddress(ctx, corporation)
	require.NoError(t, err)
	// A VSOperatorAuthorization for (co.id, grantee) blocks the operator grant.
	require.NoError(t, f.keeper.GrantVSOperatorAuthorization(ctx, co.Id, grantee,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []string{mtValidated}}))

	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem},
	})
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzExists)
}

// [MOD-DE-MSG-3] The operator path: an operator authorized for the de grant
// message can grant on the corporation's behalf; an unauthorized one cannot.
func TestGrantOperatorAuthorization_OperatorPath(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	operator := acc("operator____________")
	grantee := acc("grantee_____________")

	// Corp authorizes the operator for the de grant message (group-proposal path).
	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Operator: corporation, Grantee: operator,
		MsgTypes: []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
	})
	require.NoError(t, err)

	// The authorized operator grants to a third party on the corporation's behalf.
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Operator: operator, Grantee: grantee,
		MsgTypes: []string{mtEcosystem},
	})
	require.NoError(t, err)

	// An unauthorized operator is rejected by AUTHZ-CHECK-1.
	stranger := acc("stranger____________")
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Operator: stranger, Grantee: grantee,
		MsgTypes: []string{mtEcosystem},
	})
	require.Error(t, err)
}

func TestRevokeOperatorAuthorization(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem},
	})
	require.NoError(t, err)

	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corporation, Grantee: grantee,
	})
	require.NoError(t, err)
	list, err := qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Empty(t, list.OperatorAuthorizations)

	// Revoking a missing entry aborts.
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corporation, Grantee: grantee,
	})
	require.ErrorIs(t, err, types.ErrOperatorAuthzNotFound)
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-5/6/9] VS Operator Authorization (module-call keeper methods)
// ---------------------------------------------------------------------------

func TestGrantVSOperatorAuthorization(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	rec := func(pid uint64) types.ParticipantAuthorizationRecord {
		return types.ParticipantAuthorizationRecord{ParticipantId: pid, MsgTypes: []string{mtValidated}}
	}

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(10)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(11)))

	// Duplicate participant_id is rejected (global uniqueness).
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(10)), types.ErrParticipantRecordExists)
	// Non-delegable msg type is rejected.
	require.Error(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: []string{"/verana.de.v1.MsgUpdateParams"}}))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Len(t, vsoa.Records, 2)
	require.Equal(t, corpID, vsoa.CorporationId)
}

func TestGrantVSOperatorAuthorization_MutualExclusivity(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	// Seed an OperatorAuthorization for (corpID, vsOp).
	require.NoError(t, k.OperatorAuthorizations.Set(ctx, 99, types.OperatorAuthorization{Id: 99, CorporationId: corpID, Operator: vsOp, MsgTypes: []string{mtEcosystem}}))
	require.NoError(t, k.OperatorAuthorizationByCorpOp.Set(ctx, collections.Join(corpID, vsOp), 99))

	err := k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []string{mtValidated}})
	require.ErrorIs(t, err, types.ErrOperatorAuthzExistsMutex)
}

func TestGrantVSOperatorAuthorization_SingleCorp(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []string{mtValidated}}))
	// A different corporation cannot also authorize the same vs_operator.
	err := k.GrantVSOperatorAuthorization(ctx, 2, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 2, MsgTypes: []string{mtValidated}})
	require.ErrorIs(t, err, types.ErrVSOAOtherCorporation)
}

func TestRevokeVSOperatorAuthorization(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	mt := []string{mtValidated}
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: mt}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: mt}))
	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)

	// Removing one record leaves the VSOA in place.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 10))
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Len(t, vsoa.Records, 1)

	// Removing the last record deletes the VSOA.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 11))
	_, err = k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Error(t, err)

	// No-op when no record exists.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 999))

	// Re-granting the same pair mints a fresh vsoa id.
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: mt}))
	newID, _ := k.VSOAByParticipant.Get(ctx, 12)
	require.NotEqual(t, vsoaID, newID)
}

// [MOD-DE-MSG-5-2] fee_spend_limit is set and strictly positive iff
// with_feegrant; period requires spend_limit; expiration requires period.
func TestGrantVSOperatorAuthorization_BudgetRules(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 100))
	period := time.Hour
	now := ctx.BlockTime()

	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true,
	}), types.ErrInvalidFeeSpendLimit)
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true,
		FeeSpendLimit: sdk.Coins{sdk.NewInt64Coin("uvna", 0)},
	}), types.ErrInvalidFeeSpendLimit)
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, FeeSpendLimit: spend,
	}), types.ErrInvalidFeeSpendLimit)
	require.ErrorContains(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, Period: &period,
	}), "period requires spend_limit")
	require.ErrorContains(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, SpendLimit: spend, Expiration: &now,
	}), "expiration requires period")

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true, FeeSpendLimit: spend,
		SpendLimit: spend, Period: &period,
	}))
}

// [MOD-DE-MSG-9] Sync starts the operation cycle once, and recomputes.
func TestSyncVSOperatorAuthorization(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	period := 48 * time.Hour
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 100))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []string{mtValidated}, SpendLimit: spend, Period: &period}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []string{mtValidated}}))
	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)

	require.NoError(t, k.SyncVSOperatorAuthorization(ctx, 10))
	vsoa, _ := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NotNil(t, vsoa.Records[0].Expiration)
	require.True(t, vsoa.Records[0].Expiration.Equal(now.Add(period)))

	// A second sync leaves a started cycle alone.
	later := ctx.WithBlockTime(now.Add(time.Hour))
	require.NoError(t, k.SyncVSOperatorAuthorization(later, 10))
	vsoa, _ = k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.True(t, vsoa.Records[0].Expiration.Equal(now.Add(period)))

	// No period: no cycle.
	require.NoError(t, k.SyncVSOperatorAuthorization(ctx, 11))
	vsoa, _ = k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Nil(t, vsoa.Records[1].Expiration)

	// No-op when no record exists.
	require.NoError(t, k.SyncVSOperatorAuthorization(ctx, 999))
}

// [MOD-DE-MSG-5-5] The allowance is a PeriodicAllowance over the union of the
// live feegrant records' msg_types, capped at the sum of their fee_spend_limit.
func TestRecomputeFeeAllowance_Sum(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	f.participants.views[12] = types.ParticipantView{}             // pending
	f.participants.views[13] = types.ParticipantView{Future: true} // future counts

	fee := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("uvna", n)) }
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true, FeeSpendLimit: fee(5)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []string{mtValidated, mtCSPS}, WithFeegrant: true, FeeSpendLimit: fee(3)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: []string{mtCSPS}, WithFeegrant: true, FeeSpendLimit: fee(100)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 13, MsgTypes: []string{mtCSPS}, WithFeegrant: true, FeeSpendLimit: fee(2)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 14, MsgTypes: []string{mtCSPS}}))

	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, vsOp))
	require.NoError(t, err)
	require.Equal(t, fee(10), fg.SpendLimit)
	require.NotNil(t, fg.Period)
	require.Equal(t, types.DefaultVsOperatorFeePeriod, *fg.Period)
	require.NotNil(t, fg.Expiration)
	require.True(t, fg.Expiration.Equal(now.Add(types.DefaultVsOperatorFeePeriod)))
	require.ElementsMatch(t, []string{mtCSPS, mtValidated}, fg.MsgTypes)

	// Pending entries contribute nothing: with only 12 left, the allowance goes.
	for _, id := range []uint64{10, 11, 13, 14} {
		require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, id))
	}
	has, err := k.FeeGrants.Has(ctx, collections.Join(corpID, vsOp))
	require.NoError(t, err)
	require.False(t, has)
}

// [MOD-DE-MSG-5-5] A contributing entry with an effective_until is scheduled in
// the window-end queue; one without is not.
func TestRecomputeFeeAllowance_QueuesWindowEnd(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	until := ctx.BlockTime().Add(72 * time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until}

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []string{mtCSPS}}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []string{mtCSPS}}))

	has, err := k.WindowEndQueue.Has(ctx, collections.Join(until, uint64(10)))
	require.NoError(t, err)
	require.True(t, has)
	n := 0
	require.NoError(t, k.WindowEndQueue.Walk(ctx, nil, func(_ collections.Pair[time.Time, uint64]) (bool, error) {
		n++
		return false, nil
	}))
	require.Equal(t, 1, n)
}

func TestQueriesNotFound(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	_, err := qs.GetOperatorAuthorization(ctx, &types.QueryGetOperatorAuthorizationRequest{Id: 999})
	require.Error(t, err)
	_, err = qs.GetVSOperatorAuthorization(ctx, &types.QueryGetVSOperatorAuthorizationRequest{Id: 999})
	require.Error(t, err)
}
