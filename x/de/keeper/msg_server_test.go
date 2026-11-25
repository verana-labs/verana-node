package keeper_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
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

// testCorpID is the id mockCorpKeeper resolves every corporation account to.
const testCorpID uint64 = 1

const (
	mtEcosystem = types.MsgType_MSG_TYPE_CREATE_ECOSYSTEM
	mtSchema    = types.MsgType_MSG_TYPE_CREATE_CREDENTIAL_SCHEMA
	mtValidated = types.MsgType_MSG_TYPE_VALIDATE_ONBOARDING
	mtIssuance  = types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE
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
	msgTypes := []types.MsgType{mtEcosystem}

	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, msgTypes, nil, nil, nil))
	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, corpID, fg.GrantorCorporationId)
	require.Equal(t, grantee, fg.Grantee)
	require.Equal(t, msgTypes, fg.MsgTypes)

	// Update in place with new msg types.
	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, []types.MsgType{mtSchema}, nil, nil, nil))
	fg, err = k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, []types.MsgType{mtSchema}, fg.MsgTypes)

	require.NoError(t, k.RevokeFeeAllowance(ctx, corpID, grantee))
	has, err := k.FeeGrants.Has(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.False(t, has)
}

func TestGrantFeeAllowance_TypeURLs(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	mts := []types.MsgType{
		types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE,
		mtEcosystem,
		types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION,
	}
	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, mts, nil, nil, nil))

	fg, err := f.keeper.FeeGrants.Get(ctx, collections.Join(uint64(7), grantee))
	require.NoError(t, err)
	require.Equal(t, mts, fg.MsgTypes)

	granter, _ := sdk.AccAddressFromBech32(corpPolicyAddr(7))
	granteeAddr, _ := sdk.AccAddressFromBech32(grantee)
	a, err := f.feegrants.GetAllowance(ctx, granter, granteeAddr)
	require.NoError(t, err)
	require.Equal(t, []string{
		"/verana.pp.v1.MsgCreateOrUpdateParticipantSession",
		"/verana.ec.v1.MsgCreateEcosystem",
	}, a.(*feegrant.AllowedMsgAllowance).AllowedMessages)
}

func TestFeeAllowance_Validation(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")

	require.ErrorIs(t, k.GrantFeeAllowance(ctx, 1, grantee, []types.MsgType{types.MsgType_MSG_TYPE_UNSPECIFIED}, nil, nil, nil), types.ErrInvalidMsgType)
	require.ErrorIs(t, k.GrantFeeAllowance(ctx, 1, grantee, nil, nil, nil, nil), types.ErrInvalidMsgType)

	require.ErrorContains(t, k.RevokeFeeAllowance(ctx, 0, grantee), "grantor_corporation_id")
	require.ErrorContains(t, k.RevokeFeeAllowance(ctx, 1, ""), "grantee")
	// no-op when absent
	require.NoError(t, k.RevokeFeeAllowance(ctx, 1, acc("absent______________")))
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-3/4] Grant / Revoke Delegate Authorization (msg server)
// ---------------------------------------------------------------------------

func TestGrantDelegateAuthorization_NewAndUpdate(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)

	list, err := qs.ListDelegateAuthorizations(ctx, &types.QueryListDelegateAuthorizationsRequest{Delegate: grantee})
	require.NoError(t, err)
	require.Len(t, list.DelegateAuthorizations, 1)
	firstID := list.DelegateAuthorizations[0].Id
	require.NotZero(t, firstID)
	require.Equal(t, grantee, list.DelegateAuthorizations[0].Delegate)

	// Re-grant is an in-place update that preserves the id.
	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtSchema},
	})
	require.NoError(t, err)
	list, err = qs.ListDelegateAuthorizations(ctx, &types.QueryListDelegateAuthorizationsRequest{Delegate: grantee})
	require.NoError(t, err)
	require.Len(t, list.DelegateAuthorizations, 1)
	require.Equal(t, firstID, list.DelegateAuthorizations[0].Id)
	require.Equal(t, []types.MsgType{mtSchema}, list.DelegateAuthorizations[0].MsgTypes)

	// [MOD-DE-QRY-3] get by id.
	got, err := qs.GetDelegateAuthorization(ctx, &types.QueryGetDelegateAuthorizationRequest{Id: firstID})
	require.NoError(t, err)
	require.Equal(t, grantee, got.DelegateAuthorization.Delegate)
}

func TestGrantDelegateAuthorization_MutualExclusivity(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	// Resolve the signing corporation to its co.id (same path MSG-3 uses).
	co, err := f.corpKeeper.ResolveCorporationByPolicyAddress(ctx, corporation)
	require.NoError(t, err)
	// A VSOperatorAuthorization for (co.id, grantee) blocks the delegate grant.
	require.NoError(t, f.keeper.GrantVSOperatorAuthorization(ctx, co.Id, grantee,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []types.MsgType{mtValidated}}))

	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzExists)
}

// [MOD-DE-MSG-3] The delegate path: a delegate authorized for the de grant
// message can grant on the corporation's behalf; an unauthorized one cannot.
func TestGrantDelegateAuthorization_DelegatePath(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	delegate := acc("delegate____________")
	grantee := acc("grantee_____________")

	// Corp authorizes the delegate for the de grant message (group-proposal path).
	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: delegate,
		MsgTypes: []types.MsgType{types.MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION},
	})
	require.NoError(t, err)

	// The authorized delegate grants to a third party on the corporation's behalf.
	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: delegate, Grantee: grantee,
		MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)

	// An unauthorized delegate is rejected by AUTHZ-CHECK-1.
	stranger := acc("stranger____________")
	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: stranger, Grantee: grantee,
		MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.Error(t, err)
}

// [MOD-DE-MSG-3-2] / [MOD-DE-MSG-4-2] the group-proposal path is delegate ==
// corporation; an empty delegate is not a path.
func TestGrantDelegateAuthorization_EmptyDelegate(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.ErrorIs(t, err, types.ErrAuthzNotFound)

	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)

	_, err = ms.RevokeDelegateAuthorization(ctx, &types.MsgRevokeDelegateAuthorization{
		Corporation: corporation, Grantee: grantee,
	})
	require.ErrorIs(t, err, types.ErrAuthzNotFound)
}

// [MOD-DE-MSG-3-2] grantee MUST NOT be the corporation's own policy_address.
func TestGrantDelegateAuthorization_OwnPolicyAddress(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	delegate := acc("delegate____________")

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: delegate,
		MsgTypes: []types.MsgType{types.MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION},
	})
	require.NoError(t, err)

	for name, msg := range map[string]*types.MsgGrantDelegateAuthorization{
		"group proposal": {Corporation: corporation, Delegate: corporation, Grantee: corporation},
		"with feegrant":  {Corporation: corporation, Delegate: corporation, Grantee: corporation, WithFeegrant: true, FeegrantSpendLimit: uvna(10)},
		"delegate path":  {Corporation: corporation, Delegate: delegate, Grantee: corporation},
		"upper case":     {Corporation: corporation, Delegate: corporation, Grantee: strings.ToUpper(corporation), WithFeegrant: true},
	} {
		t.Run(name, func(t *testing.T) {
			msg.MsgTypes = []types.MsgType{mtEcosystem}
			require.NoError(t, msg.ValidateBasic())
			_, err := ms.GrantDelegateAuthorization(ctx, msg)
			require.ErrorIs(t, err, types.ErrOwnPolicyAddress)

			has, err := f.keeper.DelegateAuthorizationByCorpDelegate.Has(ctx, collections.Join(uint64(1), msg.Grantee))
			require.NoError(t, err)
			require.False(t, has)
			require.Empty(t, f.feegrants.grants)
		})
	}
}

func TestRevokeDelegateAuthorization(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)

	_, err = ms.RevokeDelegateAuthorization(ctx, &types.MsgRevokeDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee,
	})
	require.NoError(t, err)
	list, err := qs.ListDelegateAuthorizations(ctx, &types.QueryListDelegateAuthorizationsRequest{Delegate: grantee})
	require.NoError(t, err)
	require.Empty(t, list.DelegateAuthorizations)

	// Revoking a missing entry aborts.
	_, err = ms.RevokeDelegateAuthorization(ctx, &types.MsgRevokeDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee,
	})
	require.ErrorIs(t, err, types.ErrDelegateAuthzNotFound)
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
		return types.ParticipantAuthorizationRecord{ParticipantId: pid, MsgTypes: []types.MsgType{mtValidated}}
	}

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(10)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(11)))

	// Duplicate participant_id is rejected (global uniqueness).
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(10)), types.ErrParticipantRecordExists)
	// A value outside the VS account set is rejected.
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: []types.MsgType{mtEcosystem}}), types.ErrInvalidMsgType)

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
	// Seed an DelegateAuthorization for (corpID, vsOp).
	require.NoError(t, k.DelegateAuthorizations.Set(ctx, 99, types.DelegateAuthorization{Id: 99, CorporationId: corpID, Delegate: vsOp, MsgTypes: []types.MsgType{mtEcosystem}}))
	require.NoError(t, k.DelegateAuthorizationByCorpDelegate.Set(ctx, collections.Join(corpID, vsOp), 99))

	err := k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []types.MsgType{mtValidated}})
	require.ErrorIs(t, err, types.ErrDelegateAuthzExistsMutex)
}

// [MOD-DE-MSG-5-2] vs_operator MUST NOT be the corporation's own policy_address.
func TestGrantVSOperatorAuthorization_OwnPolicyAddress(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	policy := corpPolicyAddr(1)

	for name, vsOp := range map[string]string{"policy address": policy, "upper case": strings.ToUpper(policy)} {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, feegrantRecord(10, 5)), types.ErrOwnPolicyAddress)
			require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
				types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []types.MsgType{mtValidated}}), types.ErrOwnPolicyAddress)

			has, err := k.VSOAByCorpOp.Has(ctx, collections.Join(uint64(1), vsOp))
			require.NoError(t, err)
			require.False(t, has)
			require.Empty(t, f.feegrants.grants)
		})
	}

	// Another corporation's policy_address stays a valid vs_operator.
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, corpPolicyAddr(2), feegrantRecord(10, 5)))
}

func TestGrantVSOperatorAuthorization_SingleCorp(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []types.MsgType{mtValidated}}))
	corp, err := k.VSOACorpByOperator.Get(ctx, vsOp)
	require.NoError(t, err)
	require.Equal(t, uint64(1), corp)

	// A different corporation cannot also authorize the same vs_operator.
	err = k.GrantVSOperatorAuthorization(ctx, 2, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 2, MsgTypes: []types.MsgType{mtValidated}})
	require.ErrorIs(t, err, types.ErrVSOAOtherCorporation)

	// Revoking the last record frees the vs_operator for another corporation.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 1))
	_, err = k.VSOACorpByOperator.Get(ctx, vsOp)
	require.ErrorIs(t, err, collections.ErrNotFound)
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 2, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 2, MsgTypes: []types.MsgType{mtValidated}}))
}

// The single-corp check must not grow with the number of other VSOAs.
func TestGrantVSOperatorAuthorization_GasIndependentOfStoreSize(t *testing.T) {
	gasFor := func(others int) uint64 {
		f, _, ctx := setupMsgServer(t)
		k := f.keeper
		for i := 0; i < others; i++ {
			require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, acc(fmt.Sprintf("vsop_%015d", i)),
				types.ParticipantAuthorizationRecord{ParticipantId: uint64(i + 1), MsgTypes: []types.MsgType{mtValidated}}))
		}
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 2, acc("vsop_target_________"),
			types.ParticipantAuthorizationRecord{ParticipantId: 1_000_000, MsgTypes: []types.MsgType{mtValidated}}))
		return ctx.GasMeter().GasConsumed()
	}
	small := gasFor(5)
	large := gasFor(1000)
	require.Less(t, large, small*2, "gas grew with store size: %d -> %d", small, large)
}

func TestRevokeVSOperatorAuthorization(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	mt := []types.MsgType{mtValidated}
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
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true,
	}), types.ErrInvalidFeeSpendLimit)
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true,
		FeeSpendLimit: sdk.Coins{sdk.NewInt64Coin("uvna", 0)},
	}), types.ErrInvalidFeeSpendLimit)
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, FeeSpendLimit: spend,
	}), types.ErrInvalidFeeSpendLimit)
	require.ErrorContains(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, Period: &period,
	}), "period requires spend_limit")
	require.ErrorContains(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, SpendLimit: spend, Expiration: &now,
	}), "expiration requires period")

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true, FeeSpendLimit: spend,
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
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []types.MsgType{mtValidated}, SpendLimit: spend, Period: &period}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []types.MsgType{mtValidated}}))
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
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true, FeeSpendLimit: fee(5)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []types.MsgType{mtValidated, mtIssuance}, WithFeegrant: true, FeeSpendLimit: fee(3)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true, FeeSpendLimit: fee(100)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 13, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true, FeeSpendLimit: fee(2)}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 14, MsgTypes: []types.MsgType{mtIssuance}}))

	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, vsOp))
	require.NoError(t, err)
	require.Equal(t, fee(10), fg.SpendLimit)
	require.NotNil(t, fg.Period)
	require.Equal(t, types.DefaultVsOperatorFeePeriod, *fg.Period)
	require.NotNil(t, fg.Expiration)
	require.True(t, fg.Expiration.Equal(now.Add(types.DefaultVsOperatorFeePeriod)))
	require.ElementsMatch(t, []types.MsgType{mtIssuance, mtValidated}, fg.MsgTypes)

	// Pending entries contribute nothing: with only 12 left, the allowance goes.
	for _, id := range []uint64{10, 11, 13, 14} {
		require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, id))
	}
	has, err := k.FeeGrants.Has(ctx, collections.Join(corpID, vsOp))
	require.NoError(t, err)
	require.False(t, has)
}

// [MOD-DE-MSG-5-5] A feegrant entry with an effective_until is scheduled in the
// window-end queue; a non-feegrant one, or one without a window end, is not.
func TestRecomputeFeeAllowance_QueuesWindowEnd(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	until := ctx.BlockTime().Add(72 * time.Hour)
	f.participants.views[10] = types.ParticipantView{Active: true, EffectiveUntil: &until}
	f.participants.views[12] = types.ParticipantView{Active: true, EffectiveUntil: &until}
	fee := sdk.NewCoins(sdk.NewInt64Coin("uvna", 5))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true, FeeSpendLimit: fee}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: []types.MsgType{mtIssuance}, WithFeegrant: true, FeeSpendLimit: fee}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: []types.MsgType{mtIssuance}}))

	has, err := k.WindowEndQueue.Has(ctx, collections.Join(until, uint64(10)))
	require.NoError(t, err)
	require.True(t, has)
	has, err = k.WindowEndQueue.Has(ctx, collections.Join(until, uint64(12)))
	require.NoError(t, err)
	require.False(t, has, "non-feegrant records do not affect the allowance")
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
	_, err := qs.GetDelegateAuthorization(ctx, &types.QueryGetDelegateAuthorizationRequest{Id: 999})
	require.Error(t, err)
	_, err = qs.GetVSOperatorAuthorization(ctx, &types.QueryGetVSOperatorAuthorizationRequest{Id: 999})
	require.Error(t, err)
}
