package keeper_test

import (
	"errors"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/pp/keeper"
	"github.com/verana-labs/verana-node/x/pp/types"
)

type feeTxStub struct {
	sdk.FeeTx
	granter []byte
}

func (s feeTxStub) FeeGranter() []byte { return s.granter }

type plainTxStub struct{ sdk.Tx }

// anteCtx returns the context NewFeeGranterAnteHandler hands to the next handler.
func anteCtx(t *testing.T, ctx sdk.Context, tx sdk.Tx) sdk.Context {
	t.Helper()
	var got sdk.Context
	_, err := keeper.NewFeeGranterAnteHandler(func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		got = c
		return c, nil
	})(ctx, tx, false)
	require.NoError(t, err)
	return got
}

func TestFeeGranterAnteHandler_CallsNextOnce(t *testing.T) {
	_, _, ctx, _, _ := setupTriggerResolver(t)
	tx := feeTxStub{granter: sdk.AccAddress([]byte("corp_trigger________"))}
	nextErr := errors.New("next failed")

	var calls int
	var gotTx sdk.Tx
	var gotSimulate bool
	out, err := keeper.NewFeeGranterAnteHandler(func(c sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		calls++
		gotTx, gotSimulate = tx, simulate
		return c.WithBlockHeight(7), nextErr
	})(ctx, tx, true)

	require.ErrorIs(t, err, nextErr)
	require.Equal(t, 1, calls)
	require.Equal(t, tx, gotTx)
	require.True(t, gotSimulate)
	require.EqualValues(t, 7, out.BlockHeight())
}

func TestTriggerResolver_FeeGrantCheck(t *testing.T) {
	corporation := sdk.AccAddress([]byte("corp_trigger________"))
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	other := sdk.AccAddress([]byte("other_granter_______"))

	cases := []struct {
		name      string
		tx        sdk.Tx // nil: no ante handler ran
		feeErr    error
		wantCalls int
		wantErr   string
	}{
		{name: "corp pays", tx: feeTxStub{granter: corporation}, wantCalls: 1},
		{name: "corp pays, record without with_feegrant", tx: feeTxStub{granter: corporation}, feeErr: errors.New("feegrant not enabled"), wantCalls: 1, wantErr: "fee grant not authorized"},
		{name: "other granter", tx: feeTxStub{granter: other}, feeErr: errors.New("unused")},
		{name: "no granter", tx: feeTxStub{}, feeErr: errors.New("unused")},
		{name: "not a fee tx", tx: plainTxStub{}, feeErr: errors.New("unused")},
		{name: "no ante handler", feeErr: errors.New("unused")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, ctx, trk, del := setupTriggerResolver(t)
			corpID := trk.RegisterCorp(corporation.String())
			pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, operator, "did:example:holder", 0, ctx.BlockTime().Add(-time.Hour)))
			require.NoError(t, err)
			del.ErrToReturn = nil
			del.FeeGrantErr = tc.feeErr

			if tc.tx != nil {
				ctx = anteCtx(t, ctx, tc.tx)
			}
			_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{Corporation: corporation.String(), Operator: operator, Id: pid})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, del.FeeGrantCheckCalls, tc.wantCalls)
			for _, id := range del.FeeGrantCheckCalls {
				require.Equal(t, pid, id)
			}
		})
	}
}

// Path 2 (ancestor operator) has no VS-operator record, so the check never runs.
func TestTriggerResolver_FeeGrantCheckSkippedOnPath2(t *testing.T) {
	k, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________"))
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	corpID := trk.RegisterCorp(corporation.String())
	past := ctx.BlockTime().Add(-time.Hour)
	ancestorID, err := k.CreateParticipant(ctx, activeParticipant(corpID, "other_vsop", "did:example:anc", 0, past))
	require.NoError(t, err)
	pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, "someone_else", "did:example:holder", ancestorID, past))
	require.NoError(t, err)
	del.ErrToReturn = nil
	del.FeeGrantErr = errors.New("unused")

	ctx = anteCtx(t, ctx, feeTxStub{granter: corporation})
	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{Corporation: corporation.String(), Operator: operator, Id: pid})
	require.NoError(t, err)
	require.Empty(t, del.FeeGrantCheckCalls)
}

func TestSetParticipantOPToValidated_FeeGrantCheck(t *testing.T) {
	validatorCorp := sdk.AccAddress([]byte("vso_validator_______"))
	applicantAddr := sdk.AccAddress([]byte("vso_applicant_______")).String()
	vsOperator := sdk.AccAddress([]byte("vso_operator________")).String()

	cases := []struct {
		name      string
		viaVS     bool
		feeErr    error
		wantCalls int
		wantErr   string
	}{
		{name: "vs operator, corp pays", viaVS: true, wantCalls: 1},
		{name: "vs operator, record without with_feegrant", viaVS: true, feeErr: errors.New("feegrant not enabled"), wantCalls: 1, wantErr: "fee grant not authorized"},
		{name: "operator path, corp pays", feeErr: errors.New("unused")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, csKeeper, trkKeeper, goCtx, del := setupMsgServerWithDelegation(t)
			ctx := sdk.UnwrapSDKContext(goCtx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
			now := ctx.BlockTime()
			past, future := now.Add(-time.Hour), now.Add(365*24*time.Hour)
			csKeeper.CreateMockCredentialSchema(1,
				cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
				cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)

			validatorID, err := k.CreateParticipant(ctx, types.Participant{
				SchemaId: 1, Role: types.ParticipantRole_ISSUER_GRANTOR,
				CorporationId: trkKeeper.RegisterCorp(validatorCorp.String()),
				Created:       &now, Modified: &now, OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
			})
			require.NoError(t, err)
			applicantID, err := k.CreateParticipant(ctx, types.Participant{
				SchemaId: 1, Role: types.ParticipantRole_ISSUER,
				CorporationId: trkKeeper.RegisterCorp(applicantAddr),
				Created:       &now, Modified: &now, ValidatorParticipantId: validatorID, OpState: types.OnboardingState_PENDING,
			})
			require.NoError(t, err)

			if tc.viaVS {
				del.OperatorAuthErr = errors.New("operator not authorized")
			}
			del.FeeGrantErr = tc.feeErr
			ctx = anteCtx(t, ctx, feeTxStub{granter: validatorCorp})
			_, err = ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
				Corporation: validatorCorp.String(), Operator: vsOperator, Id: applicantID,
				ValidationFees: 10, IssuanceFees: 5, VerificationFees: 3,
				EffectiveUntil: &future, OpSummaryDigest: "sha384-validDigest",
			})

			p, getErr := k.GetParticipantByID(ctx, applicantID)
			require.NoError(t, getErr)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Equal(t, types.OnboardingState_PENDING, p.OpState)
			} else {
				require.NoError(t, err)
				require.Equal(t, types.OnboardingState_VALIDATED, p.OpState)
			}
			require.Len(t, del.FeeGrantCheckCalls, tc.wantCalls)
			for _, id := range del.FeeGrantCheckCalls {
				require.Equal(t, validatorID, id)
			}
		})
	}
}

func TestCreateOrUpdateParticipantSession_FeeGrantCheck(t *testing.T) {
	corporation := sdk.AccAddress([]byte("csps_corporation____"))
	operator := sdk.AccAddress([]byte("csps_operator_______")).String()
	other := sdk.AccAddress([]byte("csps_other_granter__"))

	cases := []struct {
		name      string
		verifier  bool
		granter   sdk.AccAddress
		feeErr    error
		wantCalls int
		wantErr   string
	}{
		{name: "issuer, corp pays", granter: corporation, wantCalls: 1},
		{name: "verifier, corp pays", verifier: true, granter: corporation, wantCalls: 1},
		{name: "corp pays, record without with_feegrant", granter: corporation, feeErr: errors.New("feegrant not enabled"), wantCalls: 1, wantErr: "fee grant not authorized"},
		{name: "other granter", granter: other, feeErr: errors.New("unused")},
		{name: "operator pays", feeErr: errors.New("unused")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, csKeeper, trkKeeper, goCtx, del := setupMsgServerWithDelegation(t)
			ctx := sdk.UnwrapSDKContext(goCtx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
			now := ctx.BlockTime()
			past := now.Add(-time.Hour)
			csKeeper.CreateMockCredentialSchema(1,
				cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS,
				cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_ONBOARDING_PROCESS)
			corpID := trkKeeper.RegisterCorp(corporation.String())

			ecosystemID, err := k.CreateParticipant(ctx, types.Participant{
				SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, CorporationId: corpID,
				Created: &now, Adjusted: &now, Modified: &now, OpState: types.OnboardingState_VALIDATED,
				IssuanceFees: 5, VerificationFees: 3, EffectiveFrom: &past,
			})
			require.NoError(t, err)
			newVS := func(role types.ParticipantRole) uint64 {
				id, err := k.CreateParticipant(ctx, types.Participant{
					SchemaId: 1, Role: role, CorporationId: corpID, VsOperator: operator,
					Created: &now, Adjusted: &now, Modified: &now, ValidatorParticipantId: ecosystemID,
					OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
				})
				require.NoError(t, err)
				return id
			}
			msg := &types.MsgCreateOrUpdateParticipantSession{Corporation: corporation.String(), Operator: operator, Id: uuid.New().String()}
			wantID := newVS(types.ParticipantRole_ISSUER)
			msg.IssuerParticipantId = wantID
			if tc.verifier {
				msg.IssuerParticipantId = 0
				wantID = newVS(types.ParticipantRole_VERIFIER)
				msg.VerifierParticipantId = wantID
			}

			del.FeeGrantErr = tc.feeErr
			if tc.granter != nil {
				ctx = anteCtx(t, ctx, feeTxStub{granter: tc.granter})
			}
			_, err = ms.CreateOrUpdateParticipantSession(ctx, msg)

			_, getErr := k.ParticipantSession.Get(ctx, msg.Id)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Error(t, getErr, "session must not be stored")
			} else {
				require.NoError(t, err)
				require.NoError(t, getErr)
			}
			require.Len(t, del.FeeGrantCheckCalls, tc.wantCalls)
			for _, id := range del.FeeGrantCheckCalls {
				require.Equal(t, wantID, id)
			}
		})
	}
}
