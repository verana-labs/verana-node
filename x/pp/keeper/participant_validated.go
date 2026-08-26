package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/pp/types"
)

func getValidityPeriod(participantType uint32, cs cstypes.CredentialSchema) uint32 {
	switch participantType {
	case 3: // ISSUER_GRANTOR
		return cs.IssuerGrantorValidationValidityPeriod
	case 4: // VERIFIER_GRANTOR
		return cs.VerifierGrantorValidationValidityPeriod
	case 1: // ISSUER
		return cs.IssuerValidationValidityPeriod
	case 2: // VERIFIER
		return cs.VerifierValidationValidityPeriod
	case 6: // HOLDER
		return cs.HolderValidationValidityPeriod
	default:
		return 0
	}
}

func calculateVPExp(currentVPExp *time.Time, validityPeriod uint64, now time.Time) *time.Time {
	if validityPeriod == 0 {
		return nil
	}

	var exp time.Time
	if currentVPExp == nil {
		exp = now.AddDate(0, 0, int(validityPeriod))
	} else {
		exp = currentVPExp.AddDate(0, 0, int(validityPeriod))
	}
	return &exp
}

// [MOD-PP-MSG-3-2-4] Set Participant OP to Validated overlap checks.
// Two entries of the same applicant (corporation_id, did) must not be effective
// at the same time under the same validator_participant_id for the same role.
// The entry being validated becomes effective at now, so any active sibling
// overlaps it; a future sibling overlaps unless effective_until is set and the
// sibling starts at or after it. Excluding the applicant itself is required: on
// a renewal the entry being validated is an active participant and would always
// match. schema_id is implied by validator_participant_id.
// For each, check that time ranges don't overlap.
func (ms msgServer) checkValidatedOverlap(ctx sdk.Context, applicantParticipant types.Participant, effectiveUntil *time.Time) error {
	now := ctx.BlockTime()
	var conflict types.Participant
	var found bool
	err := ms.Participant.Walk(ctx, nil, func(_ uint64, p types.Participant) (bool, error) {
		if p.Id == applicantParticipant.Id ||
			p.ValidatorParticipantId != applicantParticipant.ValidatorParticipantId ||
			p.Role != applicantParticipant.Role ||
			p.CorporationId != applicantParticipant.CorporationId ||
			p.Did != applicantParticipant.Did {
			return false, nil
		}
		if isActiveParticipant(p, now) ||
			(isFutureParticipant(p, now) && (effectiveUntil == nil || p.EffectiveFrom.Before(*effectiveUntil))) {
			conflict = p
			found = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	if found {
		return overlapError(conflict)
	}
	return nil
}

func (ms msgServer) executeSetParticipantVPToValidated(
	ctx sdk.Context,
	applicantParticipant types.Participant,
	validatorParticipant types.Participant,
	cs cstypes.CredentialSchema,
	msg *types.MsgSetParticipantOPToValidated,
	now time.Time,
	vpExp *time.Time,
	effectiveUntil *time.Time,
) (*types.MsgSetParticipantOPToValidatedResponse, error) {

	// Guard: cannot validate a slashed participant that has not been repaid
	if applicantParticipant.Slashed != nil && applicantParticipant.Repaid == nil {
		return nil, fmt.Errorf("cannot validate a slashed participant that has not been repaid")
	}

	// Update Participant applicant_participant:
	applicantParticipant.Modified = &now
	applicantParticipant.OpState = types.OnboardingState_VALIDATED
	applicantParticipant.OpLastStateChange = &now
	applicantParticipant.OpSummaryDigest = msg.OpSummaryDigest
	applicantParticipant.OpExp = vpExp
	applicantParticipant.EffectiveUntil = effectiveUntil

	// if applicant_participant.effective_from IS NULL (first time method is called for this participant, not a renewal):
	if applicantParticipant.EffectiveFrom == nil {
		applicantParticipant.ValidationFees = msg.ValidationFees
		applicantParticipant.IssuanceFees = msg.IssuanceFees
		applicantParticipant.VerificationFees = msg.VerificationFees
		applicantParticipant.IssuanceFeeDiscount = msg.IssuanceFeeDiscount
		applicantParticipant.VerificationFeeDiscount = msg.VerificationFeeDiscount
		applicantParticipant.EffectiveFrom = &now
	}
	// Renewal case: discounts are already validated to match existing, so no need to set them again

	// [MOD-PP-MSG-3-3] Fees and Trust Deposits:
	// transfer the full amount applicant_participant.op_current_fees, in the proper
	// denom (the schema pricing asset), from escrow account to validator account
	validatorCorpAcct, err := ms.corpAccountFromID(ctx, validatorParticipant.CorporationId)
	if err != nil {
		return nil, err
	}
	if applicantParticipant.OpCurrentFees > 0 {
		validatorAddr, err := sdk.AccAddressFromBech32(validatorCorpAcct)
		if err != nil {
			return nil, fmt.Errorf("invalid validator address: %w", err)
		}

		vpCurrentFeesI64, err := uint64ToInt64(applicantParticipant.OpCurrentFees, "op_current_fees")
		if err != nil {
			return nil, err
		}
		err = ms.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName,
			validatorAddr,
			sdk.NewCoins(sdk.NewInt64Coin(feeDenomForSchema(cs), vpCurrentFeesI64)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to transfer fees to validator: %w", err)
		}
	}

	// [MOD-PP-MSG-3-3] Increase validator participant trust deposit:
	// use [MOD-TD-MSG-1] to increase by applicant_participant.op_current_deposit
	if applicantParticipant.OpCurrentDeposit > 0 {
		vpCurrentDepositI64, err := uint64ToInt64(applicantParticipant.OpCurrentDeposit, "op_current_deposit")
		if err != nil {
			return nil, err
		}
		err = ms.trustDeposit.AdjustTrustDeposit(
			ctx,
			validatorCorpAcct,
			vpCurrentDepositI64,
			"participant_validated_deposit",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to adjust validator trust deposit: %w", err)
		}

		// Set applicant_participant.op_validator_deposit to applicant_participant.op_validator_deposit + applicant_participant.op_current_deposit
		applicantParticipant.OpValidatorDeposit += applicantParticipant.OpCurrentDeposit
	}

	// set applicant_participant.op_current_fees to 0
	applicantParticipant.OpCurrentFees = 0
	// set applicant_participant.op_current_deposit to 0
	applicantParticipant.OpCurrentDeposit = 0

	// Persist the updated participant
	if err := ms.Keeper.UpdateParticipant(ctx, applicantParticipant); err != nil {
		return nil, fmt.Errorf("failed to update participant: %w", err)
	}

	// [MOD-PP-MSG-3-3] Activate any disabled VSOA record by syncing its expiration
	// to the participant's effective_until via [MOD-DE-MSG-9], unconditionally: a
	// nil effective_until means the record never expires. No-op if no record.
	if err := ms.delegationKeeper.UpdateVSOperatorAuthorizationExpiration(ctx, applicantParticipant.Id, applicantParticipant.EffectiveUntil); err != nil {
		return nil, fmt.Errorf("failed to update VS operator authorization expiration: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSetParticipantOPToValidated,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(applicantParticipant.CorporationId, 10)),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorParticipantID, strconv.FormatUint(applicantParticipant.ValidatorParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyOpSummaryDigest, msg.OpSummaryDigest),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(applicantParticipant.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(msg.ValidationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyIssuanceFees, strconv.FormatUint(msg.IssuanceFees, 10)),
			sdk.NewAttribute(types.AttributeKeyVerificationFees, strconv.FormatUint(msg.VerificationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyIssuanceFeeDiscount, strconv.FormatUint(applicantParticipant.IssuanceFeeDiscount, 10)),
			sdk.NewAttribute(types.AttributeKeyVerificationFeeDiscount, strconv.FormatUint(applicantParticipant.VerificationFeeDiscount, 10)),
			sdk.NewAttribute(types.AttributeKeyOpExp, formatTimePtr(vpExp)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSetParticipantOPToValidatedResponse{}, nil
}
