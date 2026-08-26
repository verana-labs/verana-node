package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// [MOD-PP-MSG-3-2-4] validation-time overlap: scoped to (validator, role,
// corporation, did); active siblings always block, future siblings block only
// when their window intersects the requested one.
func TestSetParticipantVPToValidated_OverlapAlignment(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorAddr := sdk.AccAddress([]byte("ova_validator_addr__")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER_GRANTOR, Did: "did:example:ova-validator",
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now, Modified: &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	validate := func(id uint64, until *time.Time) error {
		_, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
			Corporation: validatorAddr, Operator: validatorAddr,
			Id: id, OpSummaryDigest: "sha384-validDigest", EffectiveUntil: until,
		})
		return err
	}
	mkParticipant := func(p types.Participant) uint64 {
		id, err := k.CreateParticipant(sdkCtx, p)
		require.NoError(t, err)
		return id
	}

	t.Run("different did sibling does not block validation", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("ova_corp_twodids____")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		activeUntil := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-svc-a",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime, EffectiveUntil: &activeUntil,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-svc-b",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		require.NoError(t, validate(applicantID, &until))
	})

	t.Run("active same-did sibling blocks", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("ova_corp_active_____")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		activeUntil := now.Add(2 * time.Hour)
		sibID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-act",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime, EffectiveUntil: &activeUntil,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-act",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		err := validate(applicantID, &until)
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("existing participant %d would be effective at the same time", sibID))
	})

	t.Run("future sibling starting before effective_until blocks", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("ova_corp_futblock___")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		sibFrom := now.Add(30 * time.Minute)
		sibUntil := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-fut",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &sibFrom, EffectiveUntil: &sibUntil,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-fut",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		err := validate(applicantID, &until)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})

	t.Run("future sibling starting at effective_until does not block", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("ova_corp_futbound___")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		until := now.Add(1 * time.Hour)
		sibUntil := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-bound",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &until, EffectiveUntil: &sibUntil,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-bound",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		require.NoError(t, validate(applicantID, &until))
	})

	t.Run("future sibling blocks a never-expiring validation", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("ova_corp_futnil_____")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		sibFrom := now.Add(30 * time.Minute)
		sibUntil := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-nil",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &sibFrom, EffectiveUntil: &sibUntil,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:ova-nil",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		// schema validity period is 0, so a nil effective_until stays unresolved
		err := validate(applicantID, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})
}

// [MOD-PP-MSG-8-2-4] adjustment overlap: the adjusted window must not overlap a
// sibling of the same (schema, role, validator, corporation, did); disjoint
// windows are allowed.
func TestSetParticipantEffectiveUntil_OverlapAlignment(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)

	adjust := func(corp string, id uint64, until time.Time) error {
		_, err := ms.SetParticipantEffectiveUntil(ctx, &types.MsgSetParticipantEffectiveUntil{
			Corporation: corp, Operator: corp, Id: id, EffectiveUntil: &until,
		})
		return err
	}
	mkParticipant := func(p types.Participant) uint64 {
		id, err := k.CreateParticipant(sdkCtx, p)
		require.NoError(t, err)
		return id
	}

	t.Run("adjustment disjoint from future sibling succeeds", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("adj_corp_disjoint___")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		u := now.Add(1 * time.Hour)
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-a",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime, EffectiveUntil: &u,
		})
		sibFrom := now.Add(2 * time.Hour)
		sibUntil := now.Add(3 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-a",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &sibFrom, EffectiveUntil: &sibUntil,
		})
		require.NoError(t, adjust(corp, applicantID, now.Add(90*time.Minute)))
	})

	t.Run("adjustment into future sibling window aborts", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("adj_corp_overlap____")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		u := now.Add(1 * time.Hour)
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-b",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime, EffectiveUntil: &u,
		})
		sibFrom := now.Add(2 * time.Hour)
		sibUntil := now.Add(3 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-b",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &sibFrom, EffectiveUntil: &sibUntil,
		})
		err := adjust(corp, applicantID, now.Add(150*time.Minute))
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})

	t.Run("different did sibling is ignored", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("adj_corp_otherdid___")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		u := now.Add(1 * time.Hour)
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-c",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime, EffectiveUntil: &u,
		})
		sibFrom := now.Add(2 * time.Hour)
		sibUntil := now.Add(3 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-other",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &sibFrom, EffectiveUntil: &sibUntil,
		})
		require.NoError(t, adjust(corp, applicantID, now.Add(150*time.Minute)))
	})

	t.Run("never-expiring sibling in the adjusted window aborts with remedy", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("adj_corp_neverexp___")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		u := now.Add(1 * time.Hour)
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-d",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime, EffectiveUntil: &u,
		})
		sibFrom := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:adj-d",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &sibFrom,
		})
		err := adjust(corp, applicantID, now.Add(150*time.Minute))
		require.Error(t, err)
		require.Contains(t, err.Error(), "never expires")
		require.Contains(t, err.Error(), "SetParticipantEffectiveUntil")
	})
}

// [MOD-PP-MSG-14-2-4] self-create overlap: scoped to (validator, role,
// corporation, did); a second service self-creates freely, disjoint successor
// windows are allowed, a never-expiring sibling points at the remedy.
func TestSelfCreateParticipant_OverlapAlignment(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	authority := sdk.AccAddress([]byte("sca_authority_______")).String()
	validDid := "did:example:sca-eco"
	trID := trkKeeper.CreateMockEcosystem(authority, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)
	corpID := trkKeeper.RegisterCorp(authority)
	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: validDid,
		CorporationId: corpID, Created: &now, Modified: &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	selfCreate := func(did string, from *time.Time, until *time.Time) error {
		_, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation: authority, Operator: authority,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorID,
			Did:                    did,
			EffectiveFrom:          from, EffectiveUntil: until,
		})
		return err
	}

	activeUntil := now.Add(1 * time.Hour)
	_, err = k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:sca-svc-a",
		CorporationId: corpID, Created: &now, Modified: &now,
		ValidatorParticipantId: validatorID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime, EffectiveUntil: &activeUntil,
	})
	require.NoError(t, err)

	t.Run("second service self-creates while first is active", func(t *testing.T) {
		u := now.Add(1 * time.Hour)
		require.NoError(t, selfCreate("did:example:sca-svc-b", nil, &u))
	})

	t.Run("active same-did sibling blocks", func(t *testing.T) {
		u := now.Add(1 * time.Hour)
		err := selfCreate("did:example:sca-svc-a", nil, &u)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})

	t.Run("disjoint successor window succeeds", func(t *testing.T) {
		from := now.Add(2 * time.Hour)
		until := now.Add(3 * time.Hour)
		require.NoError(t, selfCreate("did:example:sca-svc-a", &from, &until))
	})

	t.Run("window overlapping a future sibling aborts", func(t *testing.T) {
		from := now.Add(150 * time.Minute)
		until := now.Add(4 * time.Hour)
		err := selfCreate("did:example:sca-svc-a", &from, &until)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})

	t.Run("never-expiring same-did sibling aborts with remedy", func(t *testing.T) {
		did := "did:example:sca-svc-c"
		_, err := k.CreateParticipant(sdkCtx, types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: did,
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		})
		require.NoError(t, err)
		from := now.Add(2 * time.Hour)
		until := now.Add(3 * time.Hour)
		err = selfCreate(did, &from, &until)
		require.Error(t, err)
		require.Contains(t, err.Error(), "never expires")
		require.Contains(t, err.Error(), "SetParticipantEffectiveUntil")
	})
}

// Full lifecycle across MSG-1 and MSG-3: a corporation onboards two services
// with one validator, both validate, and a successor for the first service
// validates only after the current entry expires.
func TestTwoServicesEndToEnd(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorAddr := sdk.AccAddress([]byte("e2e_validator_addr__")).String()
	authority := sdk.AccAddress([]byte("e2e_authority_______")).String()
	trID := trkKeeper.CreateMockEcosystem(validatorAddr, "did:example:e2e-eco")
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS)
	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:e2e-eco",
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now, Modified: &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)
	trkKeeper.RegisterCorp(authority)

	startOP := func(did string) (uint64, error) {
		resp, err := ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
			Corporation: authority, Operator: authority,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorID,
			Did:                    did,
		})
		if err != nil {
			return 0, err
		}
		return resp.ParticipantId, nil
	}
	validate := func(c sdk.Context, id uint64, until time.Time) error {
		_, err := ms.SetParticipantOPToValidated(sdk.WrapSDKContext(c), &types.MsgSetParticipantOPToValidated{
			Corporation: validatorAddr, Operator: validatorAddr,
			Id: id, OpSummaryDigest: "sha384-validDigest", EffectiveUntil: &until,
		})
		return err
	}

	svcAUntil := now.Add(1 * time.Hour)

	// service A onboards and validates
	aID, err := startOP("did:example:e2e-svc-a")
	require.NoError(t, err)
	require.NoError(t, validate(sdkCtx, aID, svcAUntil))

	// service B onboards with the same validator and validates too
	bID, err := startOP("did:example:e2e-svc-b")
	require.NoError(t, err)
	require.NoError(t, validate(sdkCtx, bID, now.Add(1*time.Hour)))

	a, err := k.GetParticipantByID(sdkCtx, aID)
	require.NoError(t, err)
	b, err := k.GetParticipantByID(sdkCtx, bID)
	require.NoError(t, err)
	require.Equal(t, types.OnboardingState_VALIDATED, a.OpState)
	require.Equal(t, types.OnboardingState_VALIDATED, b.OpState)

	// the successor process for service A starts while A is still active
	succID, err := startOP("did:example:e2e-svc-a")
	require.NoError(t, err)

	// but it cannot validate until A's window ends
	err = validate(sdkCtx, succID, now.Add(3*time.Hour))
	require.Error(t, err)
	require.Contains(t, err.Error(), "would be effective at the same time")

	// after A expires the successor validates
	laterCtx := sdkCtx.WithBlockTime(svcAUntil.Add(1 * time.Minute))
	require.NoError(t, validate(laterCtx, succID, svcAUntil.Add(2*time.Hour)))
}

// Entry states outside the active and future definitions never participate in
// overlap checks, and boundary windows do not overlap.
func TestOverlapChecks_EntryStates(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorAddr := sdk.AccAddress([]byte("pin_validator_addr__")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now, Modified: &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	validate := func(id uint64, until *time.Time) error {
		_, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
			Corporation: validatorAddr, Operator: validatorAddr,
			Id: id, OpSummaryDigest: "sha384-validDigest", EffectiveUntil: until,
		})
		return err
	}
	mkParticipant := func(p types.Participant) uint64 {
		id, err := k.CreateParticipant(sdkCtx, p)
		require.NoError(t, err)
		return id
	}

	// a PENDING sibling has no effective_from, so it is neither active nor future
	t.Run("MSG-3 pending nil-from sibling does not block", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("pin_corp_m6a________")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin6a",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin6a",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		require.NoError(t, validate(applicantID, &until))
	})

	t.Run("MSG-8 pending nil-from sibling does not block", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("pin_corp_m6c________")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		u := now.Add(1 * time.Hour)
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:pin6c",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime, EffectiveUntil: &u,
		})
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:pin6c",
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState: types.OnboardingState_PENDING,
		})
		newUntil := now.Add(2 * time.Hour)
		_, err := ms.SetParticipantEffectiveUntil(ctx, &types.MsgSetParticipantEffectiveUntil{
			Corporation: corp, Operator: corp, Id: applicantID, EffectiveUntil: &newUntil,
		})
		require.NoError(t, err)
	})

	// effective_until is exclusive: a window starting exactly at another's end
	// does not overlap it
	t.Run("MSG-7 new window starting exactly at existing effective_until succeeds", func(t *testing.T) {
		authority := sdk.AccAddress([]byte("pin_root_auth_______")).String()
		trID := trkKeeper.CreateMockEcosystem(authority, "did:example:pinroot")
		csKeeper.UpdateMockCredentialSchema(11, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
		firstFrom := now.Add(1 * time.Hour)
		firstUntil := now.Add(5 * time.Hour)
		_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: authority,
			SchemaId: 11, Did: "did:example:pinroot",
			EffectiveFrom: &firstFrom, EffectiveUntil: &firstUntil,
		})
		require.NoError(t, err)
		secondUntil := now.Add(10 * time.Hour)
		_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: authority,
			SchemaId: 11, Did: "did:example:pinroot",
			EffectiveFrom: &firstUntil, EffectiveUntil: &secondUntil,
		})
		require.NoError(t, err)
	})

	// an entry whose window is empty (from == until) is neither active nor future
	t.Run("MSG-3 empty-window sibling does not block", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("pin_corp_m8_________")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		emptyPoint := now.Add(30 * time.Minute)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin8",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &emptyPoint, EffectiveUntil: &emptyPoint,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin8",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		require.NoError(t, validate(applicantID, &until))
	})

	// the glossary definitions carry no repaid term: a repaid sibling still
	// blocks on both the active and the future leg
	t.Run("MSG-3 repaid active sibling still blocks", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("pin_corp_m9_________")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		repaidAt := now.Add(-30 * time.Minute)
		sibUntil := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin9",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime, EffectiveUntil: &sibUntil,
			Repaid: &repaidAt,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin9",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		err := validate(applicantID, &until)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})

	t.Run("MSG-3 repaid future sibling still blocks", func(t *testing.T) {
		corp := sdk.AccAddress([]byte("pin_corp_m9f________")).String()
		corpID := trkKeeper.RegisterCorp(corp)
		repaidAt := now.Add(-30 * time.Minute)
		sibFrom := now.Add(30 * time.Minute)
		sibUntil := now.Add(2 * time.Hour)
		mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin9f",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &sibFrom, EffectiveUntil: &sibUntil,
			Repaid: &repaidAt,
		})
		applicantID := mkParticipant(types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin9f",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		until := now.Add(1 * time.Hour)
		err := validate(applicantID, &until)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would be effective at the same time")
	})
}

// The create paths ignore entries that are neither active nor future too.
func TestOverlapChecks_EntryStates_CreatePaths(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	authority := sdk.AccAddress([]byte("pin_cp_authority____")).String()
	validDid := "did:example:pin_cp"
	trID := trkKeeper.CreateMockEcosystem(authority, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)
	corpID := trkKeeper.RegisterCorp(authority)

	t.Run("MSG-7 nil-from ecosystem entry does not block", func(t *testing.T) {
		_, err := k.CreateParticipant(sdkCtx, types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: validDid,
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState: types.OnboardingState_PENDING,
		})
		require.NoError(t, err)
		until := now.Add(2 * time.Hour)
		_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: authority,
			SchemaId: 1, Did: validDid, EffectiveUntil: &until,
		})
		require.NoError(t, err)
	})

	t.Run("MSG-14 pending nil-from sibling does not block", func(t *testing.T) {
		csKeeper.UpdateMockCredentialSchema(2, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)
		validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
			SchemaId: 2, Role: types.ParticipantRole_ECOSYSTEM, Did: validDid,
			CorporationId: corpID, Created: &now, Modified: &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		})
		require.NoError(t, err)
		_, err = k.CreateParticipant(sdkCtx, types.Participant{
			SchemaId: 2, Role: types.ParticipantRole_ISSUER, Did: "did:example:pin_cp_svc",
			CorporationId: corpID, Created: &now, Modified: &now,
			ValidatorParticipantId: validatorID,
			OpState:                types.OnboardingState_PENDING,
		})
		require.NoError(t, err)
		until := now.Add(2 * time.Hour)
		_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation: authority, Operator: authority,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorID,
			Did:                    "did:example:pin_cp_svc",
			EffectiveUntil:         &until,
		})
		require.NoError(t, err)
	})
}
