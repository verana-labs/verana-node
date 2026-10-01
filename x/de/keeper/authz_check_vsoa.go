package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// emitVSOperatorAuthzUpdated signals that an AUTHZ-CHECK-3 path mutated the
// participant's authorization record (spend debit or cycle reset). It carries
// the vsoa id and participant id so the indexer can identify the record and
// re-read authoritative state via ABCI at this height.
func (k Keeper) emitVSOperatorAuthzUpdated(ctx context.Context, vsoaID, participantID uint64) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeVSOperatorAuthorizationUpdated,
			sdk.NewAttribute(types.AttributeKeyVsoaID, strconv.FormatUint(vsoaID, 10)),
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(participantID, 10)),
		),
	)
}

// CheckVSOperatorAuthorizationOnParticipant implements [AUTHZ-CHECK-3]. Callers
// MUST resolve the signing corporation account to its co.id via AUTHZ-CHECK-5
// and pass corporationID, not the signing account. The spend debit (step 6) is
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

	// 1. The Participant entry MUST be an active participant.
	pv, found := k.participantKeeper().ViewParticipant(ctx, participantID)
	if !found || !pv.Active {
		return types.ErrParticipantNotActive
	}

	// 2. Record MUST exist for participant_id.
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	// 3. Record MUST belong to VSOperatorAuthorization[corporationID, operator].
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

	// 4. msg_type MUST be in record.msg_types.
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

	// 5. Operation-budget cycle. Never aborts: expiration is the cycle clock,
	// the entry window is enforced by step 1.
	if rec.Expiration != nil && rec.Period != nil && *rec.Period > 0 && !rec.Expiration.After(now) {
		if len(rec.SpendLimit) > 0 {
			rec.RemainingSpend = rec.SpendLimit
		}
		newExp := now.Add(*rec.Period)
		rec.Expiration = &newExp
		if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
			return fmt.Errorf("failed to persist cycle reset: %w", err)
		}
		k.emitVSOperatorAuthzUpdated(ctx, vsoaID, participantID)
	}

	// 6. spend_limit deduction: ConsumeRecordSpend, called by the handler.
	return nil
}

// ConsumeRecordSpend implements [AUTHZ-CHECK-3] step 6: debits the participant's
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
	k.emitVSOperatorAuthzUpdated(ctx, vsoaID, participantID)
	return nil
}

// CheckVSOperatorFeeGrant implements [AUTHZ-CHECK-4]: the record MUST enable
// corporation-paid fees. The fee cap itself is enforced by the aggregate
// x/feegrant allowance at fee-processing time.
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
			return nil
		}
	}
	return types.ErrVSOperatorAuthzNotFound
}
