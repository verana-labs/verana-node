package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// CheckVSOperatorAuthorizationOnParticipant implements [AUTHZ-CHECK-3]. Callers
// MUST resolve the signing corporation account to its co.id via AUTHZ-CHECK-5
// and pass corporationID, not the signing account. The spend debit (step 5) is
// applied separately by ConsumeRecordSpend, once the caller knows the amount.
func (k Keeper) CheckVSOperatorAuthorizationOnParticipant(
	ctx context.Context,
	corporationID uint64,
	operator string,
	participantID uint64,
	msgType string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// 1. Record MUST exist for participant_id.
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	// 2. Record MUST belong to VSOperatorAuthorization[corporationID, operator].
	if vsoa.CorporationId != corporationID || vsoa.VsOperator != operator {
		return types.ErrVSOperatorAuthzNotFound
	}

	idx := -1
	for i := range vsoa.Records {
		if vsoa.Records[i].ParticipantId == participantID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return types.ErrVSOperatorAuthzNotFound
	}
	rec := &vsoa.Records[idx]

	// 3. msg_type MUST be in record.msg_types.
	authorized := false
	for _, mt := range rec.MsgTypes {
		if mt == msgType {
			authorized = true
			break
		}
	}
	if !authorized {
		return fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgType)
	}

	// 4. Cycle / expiration. A nil expiration means the record never expires;
	// a concrete expiration <= now is dead (the disabled state set at MOD-PP-MSG-1
	// uses a concrete now, so it stays dead here).
	if rec.Period != nil && *rec.Period > 0 && rec.Expiration != nil && !rec.Expiration.After(now) {
		if len(rec.SpendLimit) > 0 {
			rec.RemainingSpend = rec.SpendLimit
		}
		if len(rec.FeeSpendLimit) > 0 {
			rec.RemainingFeeSpend = rec.FeeSpendLimit
		}
		newExp := now.Add(*rec.Period)
		rec.Expiration = &newExp
		if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
			return fmt.Errorf("failed to persist cycle reset: %w", err)
		}
	} else if rec.Expiration != nil && !rec.Expiration.After(now) {
		return types.ErrAuthzExpired
	}

	// 5. spend_limit deduction: ConsumeRecordSpend, called by the CSPS handler.
	return nil
}

// ConsumeRecordSpend implements [AUTHZ-CHECK-3] step 5: debits the participant's
// VSOA record remaining_spend by `amount`. No-op when no spend_limit or amount is zero.
func (k Keeper) ConsumeRecordSpend(
	ctx context.Context,
	corporationID uint64,
	operator string,
	participantID uint64,
	amount sdk.Coins,
) error {
	if amount.IsZero() {
		return nil
	}
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}
	if vsoa.CorporationId != corporationID || vsoa.VsOperator != operator {
		return types.ErrVSOperatorAuthzNotFound
	}
	idx := -1
	for i := range vsoa.Records {
		if vsoa.Records[i].ParticipantId == participantID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return types.ErrVSOperatorAuthzNotFound
	}
	rec := &vsoa.Records[idx]
	if len(rec.SpendLimit) == 0 {
		return nil
	}
	if !rec.RemainingSpend.IsAllGTE(amount) {
		return fmt.Errorf("%w: spend %s exceeds remaining %s",
			types.ErrAuthzSpendLimitExceeded, amount.String(), rec.RemainingSpend.String())
	}
	rec.RemainingSpend = rec.RemainingSpend.Sub(amount...)
	if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
		return fmt.Errorf("failed to debit record spend: %w", err)
	}
	return nil
}

// ConsumeRecordFeeSpend implements [AUTHZ-CHECK-4] step 3: debits the record's remaining_fee_spend by `fee`.
func (k Keeper) ConsumeRecordFeeSpend(
	ctx context.Context,
	corporationID uint64,
	operator string,
	participantID uint64,
	fee sdk.Coins,
) error {
	if fee.IsZero() {
		return nil
	}
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}
	if vsoa.CorporationId != corporationID || vsoa.VsOperator != operator {
		return types.ErrVSOperatorAuthzNotFound
	}
	idx := -1
	for i := range vsoa.Records {
		if vsoa.Records[i].ParticipantId == participantID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return types.ErrVSOperatorAuthzNotFound
	}
	rec := &vsoa.Records[idx]
	if !rec.WithFeegrant || len(rec.FeeSpendLimit) == 0 {
		return nil
	}
	if !rec.RemainingFeeSpend.IsAllGTE(fee) {
		return fmt.Errorf("%w: fee %s exceeds remaining %s",
			types.ErrAuthzSpendLimitExceeded, fee.String(), rec.RemainingFeeSpend.String())
	}
	rec.RemainingFeeSpend = rec.RemainingFeeSpend.Sub(fee...)
	if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
		return fmt.Errorf("failed to debit record fee spend: %w", err)
	}
	return nil
}

// CheckVSOperatorFeeGrant implements [AUTHZ-CHECK-4]. It uses the same record
// lookup as AUTHZ-CHECK-3.
//
//  1. record.with_feegrant MUST be true.
//  2. The cycle / expiration reset is handled by AUTHZ-CHECK-3 (run first).
//  3. If fee_spend_limit is set, remaining_fee_spend MUST cover the estimated tx
//     fees and is deducted after success. The keeper has no visibility into the
//     fee-payment mode or amount (ante context), so the amount-based check is
//     deferred to the ante handler (matches existing csps / trigger_resolver).
func (k Keeper) CheckVSOperatorFeeGrant(ctx context.Context, participantID uint64) error {
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	for i := range vsoa.Records {
		if vsoa.Records[i].ParticipantId == participantID {
			if !vsoa.Records[i].WithFeegrant {
				return types.ErrVSOFeegrantNotEnabled
			}
			// fee_spend_limit amount check deferred to ante handler.
			return nil
		}
	}
	return types.ErrVSOperatorAuthzNotFound
}
