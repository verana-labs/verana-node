package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/pp/keeper"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// The (did, corporation_id) consistency invariant (spec MOD-PP-MSG-1-2-1 /
// 7-2-1 / 14-2-1): all Participant entries sharing a did MUST belong to the
// same corporation. Enforced at create time in the three create paths.

func TestStartParticipantOP_DIDCorporationConsistency(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_start_corp_a____")).String()
	corpB := sdk.AccAddress([]byte("did_start_corp_b____")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
	validatorID := vsoaValidator(t, k, sdkCtx, trkKeeper, corpA, now, past)

	// corpB already controls sharedDID.
	sharedDID := "did:example:start-shared"
	corpBID := trkKeeper.RegisterCorp(corpB)
	_, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: sharedDID,
		CorporationId: corpBID, Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)
	delKeeper.Reset()

	// Bad path: corpA tries to start a participant under a did corpB controls.
	_, err = ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)

	// Happy path: a did not controlled by anyone else is accepted.
	_, err = ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: "did:example:start-fresh",
	})
	require.NoError(t, err)
}

// Cross-type: a participant may not claim a did owned by another corporation's
// ECOSYSTEM (no participant shares it), per the global DID ownership invariant.
func TestStartParticipantOP_RejectsDIDOwnedByForeignEcosystem(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_ppeco_corp_a____")).String()
	corpB := sdk.AccAddress([]byte("did_ppeco_corp_b____")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
	validatorID := vsoaValidator(t, k, sdkCtx, trkKeeper, corpA, now, past)

	// corpB controls an ecosystem claiming sharedDID (no participant does).
	sharedDID := "did:example:ppeco-shared"
	trkKeeper.CreateMockEcosystem(corpB, sharedDID)
	delKeeper.Reset()

	_, err := ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

func TestCreateRootParticipant_DIDCorporationConsistency(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(24 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_root_corp_a_____")).String()
	corpB := sdk.AccAddress([]byte("did_root_corp_b_____")).String()
	ecoDID := "did:example:root-ecosystem"
	trID := trkKeeper.CreateMockEcosystem(corpA, ecoDID)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)

	// corpB already controls sharedDID.
	sharedDID := "did:example:root-shared"
	corpBID := trkKeeper.RegisterCorp(corpB)
	_, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: sharedDID,
		CorporationId: corpBID, Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)

	// Bad path: corpA creates a root participant under a did corpB controls.
	_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: corpA, Operator: corpA, SchemaId: 1, Did: sharedDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)

	// Happy path: corpA creates a root participant under its own ecosystem did.
	_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: corpA, Operator: corpA, SchemaId: 1, Did: ecoDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.NoError(t, err)
}

func TestSelfCreateParticipant_DIDCorporationConsistency(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(360 * 24 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_self_corp_a_____")).String()
	corpB := sdk.AccAddress([]byte("did_self_corp_b_____")).String()
	ecoDID := "did:example:self-ecosystem"
	trID := trkKeeper.CreateMockEcosystem(corpA, ecoDID)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

	// Active ECOSYSTEM validator participant under corpA.
	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: ecoDID,
		CorporationId: trkKeeper.RegisterCorp(corpA), Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)

	// corpB already controls sharedDID.
	sharedDID := "did:example:self-shared"
	corpBID := trkKeeper.RegisterCorp(corpB)
	_, err = k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: sharedDID,
		CorporationId: corpBID, Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)
	delKeeper.Reset()

	// Bad path: corpA self-creates under a did corpB controls.
	_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)

	// Happy path: a fresh did is accepted.
	_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: "did:example:self-fresh",
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.NoError(t, err)
}

// Cross-type: CreateRootParticipant may not claim a did owned by another
// corporation's ecosystem (no participant shares it).
func TestCreateRootParticipant_RejectsDIDOwnedByForeignEcosystem(t *testing.T) {
	_, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(24 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_root_eco_corp_a_")).String()
	corpB := sdk.AccAddress([]byte("did_root_eco_corp_b_")).String()
	ecoDID := "did:example:root-eco-self"
	trID := trkKeeper.CreateMockEcosystem(corpA, ecoDID)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)

	// corpB controls a different ecosystem claiming sharedDID.
	sharedDID := "did:example:root-eco-shared"
	trkKeeper.CreateMockEcosystem(corpB, sharedDID)

	_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: corpA, Operator: corpA, SchemaId: 1, Did: sharedDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

// Cross-type: SelfCreateParticipant may not claim a did owned by another
// corporation's ecosystem.
func TestSelfCreateParticipant_RejectsDIDOwnedByForeignEcosystem(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(360 * 24 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_self_eco_corp_a_")).String()
	corpB := sdk.AccAddress([]byte("did_self_eco_corp_b_")).String()
	ecoDID := "did:example:self-eco-self"
	trID := trkKeeper.CreateMockEcosystem(corpA, ecoDID)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: ecoDID,
		CorporationId: trkKeeper.RegisterCorp(corpA), Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)

	// corpB controls an ecosystem claiming sharedDID.
	sharedDID := "did:example:self-eco-shared"
	trkKeeper.CreateMockEcosystem(corpB, sharedDID)
	delKeeper.Reset()

	_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

// The corporation leg of assertNoForeignDIDOwnerPP: rejects a did owned by a
// foreign Corporation.did. Uses ParticipantKeeper directly to reach the corp
// mock (setupMsgServer* discards it). Fails if that branch is removed.
func TestStartParticipantOP_RejectsDIDOwnedByForeignCorporation(t *testing.T) {
	k, csKeeper, trkKeeper, coKeeper, ctx, delKeeper := keepertest.ParticipantKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	sdkCtx := ctx.WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_corp_leg_corp_a_")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
	validatorID := vsoaValidator(t, k, sdkCtx, trkKeeper, corpA, now, past)

	// A Corporation (id 99) owns sharedDID as its own did.
	sharedDID := "did:example:corp-owned"
	coKeeper.RegisterCorpDID(sharedDID, 99)
	delKeeper.Reset()

	_, err := ms.StartParticipantOP(sdkCtx, &types.MsgStartParticipantOP{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}
