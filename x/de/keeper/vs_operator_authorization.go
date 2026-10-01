package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantVSOperatorAuthorization implements [MOD-DE-MSG-5]. It is a module-call
// only method: the caller (PP lifecycle handlers) resolves co.id via
// AUTHZ-CHECK-5 and passes the full ParticipantAuthorizationRecord. No
// Participant state is read here; the recompute reads the participant view.
func (k Keeper) GrantVSOperatorAuthorization(
	ctx context.Context,
	corporationID uint64,
	vsOperator string,
	record types.ParticipantAuthorizationRecord,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// [MOD-DE-MSG-5-2] Basic checks.
	if corporationID == 0 {
		return fmt.Errorf("corporation_id cannot be 0")
	}
	if vsOperator == "" {
		return fmt.Errorf("vs_operator cannot be empty")
	}
	if len(record.SpendLimit) > 0 && !record.SpendLimit.IsValid() {
		return fmt.Errorf("invalid spend_limit")
	}
	if record.WithFeegrant {
		if len(record.FeeSpendLimit) == 0 || !record.FeeSpendLimit.IsValid() || !record.FeeSpendLimit.IsAllPositive() {
			return types.ErrInvalidFeeSpendLimit
		}
	} else if len(record.FeeSpendLimit) > 0 {
		return types.ErrInvalidFeeSpendLimit
	}
	if record.Period != nil {
		if *record.Period <= 0 {
			return fmt.Errorf("period must be a positive duration")
		}
		if len(record.SpendLimit) == 0 {
			return fmt.Errorf("period requires spend_limit")
		}
	}
	if record.Expiration != nil && record.Period == nil {
		return fmt.Errorf("expiration requires period")
	}

	// record.participant_id MUST NOT already exist anywhere (global uniqueness).
	if record.ParticipantId == 0 {
		return fmt.Errorf("record participant_id cannot be 0")
	}
	has, err := k.VSOAByParticipant.Has(ctx, record.ParticipantId)
	if err != nil {
		return fmt.Errorf("failed to check participant index: %w", err)
	}
	if has {
		return types.ErrParticipantRecordExists
	}

	// record.msg_types non-empty and only VPR delegable types.
	if len(record.MsgTypes) == 0 {
		return fmt.Errorf("record msg_types must not be empty")
	}
	for _, mt := range record.MsgTypes {
		if !types.VPRDelegableMsgTypes[mt] {
			return fmt.Errorf("%w: %s", types.ErrInvalidMsgType, mt)
		}
	}

	// Mutual exclusivity: no OperatorAuthorization may exist for
	// (corporation_id, vs_operator).
	hasOA, err := k.OperatorAuthorizationByCorpOp.Has(ctx, collections.Join(corporationID, vsOperator))
	if err != nil {
		return fmt.Errorf("failed to check OperatorAuthorization index: %w", err)
	}
	if hasOA {
		return types.ErrOperatorAuthzExistsMutex
	}

	// Single-corp constraint: no other VSOperatorAuthorization may exist where
	// vs_operator == vsOperator AND corporation_id != corporationID.
	var conflict bool
	if err := k.VSOperatorAuthorizations.Walk(ctx, nil, func(_ uint64, v types.VSOperatorAuthorization) (bool, error) {
		if v.VsOperator == vsOperator && v.CorporationId != corporationID {
			conflict = true
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("failed to scan VSOperatorAuthorizations: %w", err)
	}
	if conflict {
		return types.ErrVSOAOtherCorporation
	}

	// [MOD-DE-MSG-5-4] Execution: load or create the VSOperatorAuthorization for
	// (corporation_id, vs_operator); a new entry gets a fresh vsoa.id, an
	// existing one preserves it.
	vsoa, found, err := k.getVSOAByCorpOp(ctx, corporationID, vsOperator)
	if err != nil {
		return err
	}
	if !found {
		id, err := k.nextVSOAID(ctx)
		if err != nil {
			return err
		}
		vsoa = types.VSOperatorAuthorization{
			Id:            id,
			CorporationId: corporationID,
			VsOperator:    vsOperator,
		}
	}

	if len(record.SpendLimit) > 0 {
		record.RemainingSpend = record.SpendLimit
	}

	vsoa.Records = append(vsoa.Records, record)

	if err := k.VSOperatorAuthorizations.Set(ctx, vsoa.Id, vsoa); err != nil {
		return fmt.Errorf("failed to set VSOperatorAuthorization: %w", err)
	}
	if err := k.VSOAByCorpOp.Set(ctx, collections.Join(corporationID, vsOperator), vsoa.Id); err != nil {
		return fmt.Errorf("failed to set VSOA index: %w", err)
	}
	if err := k.VSOAByParticipant.Set(ctx, record.ParticipantId, vsoa.Id); err != nil {
		return fmt.Errorf("failed to set participant index: %w", err)
	}

	// [MOD-DE-MSG-5-5] Recompute the chain-level fee allowance.
	if err := k.recomputeFeeAllowance(ctx, vsoa); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyVsoaID, strconv.FormatUint(vsoa.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(corporationID, 10)),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsOperator),
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(record.ParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, sdkCtx.BlockTime().String()),
		),
	)
	return nil
}

// RevokeVSOperatorAuthorization implements [MOD-DE-MSG-6]. Locates the unique
// ParticipantAuthorizationRecord by participant_id and removes it; a no-op when
// no record exists. The VSOperatorAuthorization is deleted when its last record
// is removed (a later grant for the same (corp_id, vs_operator) mints a fresh
// vsoa.id).
func (k Keeper) RevokeVSOperatorAuthorization(ctx context.Context, participantID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// No record for this participant — no-op.
			return nil
		}
		return fmt.Errorf("failed to read participant index %d: %w", participantID, err)
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	newRecords := make([]types.ParticipantAuthorizationRecord, 0, len(vsoa.Records))
	for _, r := range vsoa.Records {
		if r.ParticipantId != participantID {
			newRecords = append(newRecords, r)
		}
	}
	vsoa.Records = newRecords

	if err := k.VSOAByParticipant.Remove(ctx, participantID); err != nil {
		return fmt.Errorf("failed to remove participant index: %w", err)
	}

	if len(vsoa.Records) == 0 {
		if err := k.VSOperatorAuthorizations.Remove(ctx, vsoaID); err != nil {
			return fmt.Errorf("failed to remove VSOperatorAuthorization: %w", err)
		}
		if err := k.VSOAByCorpOp.Remove(ctx, collections.Join(vsoa.CorporationId, vsoa.VsOperator)); err != nil {
			return fmt.Errorf("failed to remove VSOA index: %w", err)
		}
	} else {
		if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
			return fmt.Errorf("failed to update VSOperatorAuthorization: %w", err)
		}
	}

	// [MOD-DE-MSG-5-5] Recompute the chain-level fee allowance (revokes it when
	// no with_feegrant record remains).
	if err := k.recomputeFeeAllowance(ctx, vsoa); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyVsoaID, strconv.FormatUint(vsoaID, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(vsoa.CorporationId, 10)),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsoa.VsOperator),
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(participantID, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, sdkCtx.BlockTime().String()),
		),
	)
	return nil
}

// SyncVSOperatorAuthorization implements [MOD-DE-MSG-9]. Starts the
// operation-budget cycle of the record on first activation and recomputes the
// aggregate fee allowance of its VSOA; a no-op when no record exists.
func (k Keeper) SyncVSOperatorAuthorization(ctx context.Context, participantID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// No record for this participant — no-op.
			return nil
		}
		return fmt.Errorf("failed to read participant index %d: %w", participantID, err)
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	for i := range vsoa.Records {
		rec := &vsoa.Records[i]
		if rec.ParticipantId != participantID {
			continue
		}
		if rec.Period != nil && rec.Expiration == nil {
			exp := now.Add(*rec.Period)
			rec.Expiration = &exp
			if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
				return fmt.Errorf("failed to update VSOperatorAuthorization: %w", err)
			}
		}
		break
	}

	if err := k.recomputeFeeAllowance(ctx, vsoa); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyVsoaID, strconv.FormatUint(vsoaID, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(vsoa.CorporationId, 10)),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsoa.VsOperator),
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(participantID, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)
	return nil
}

// recomputeFeeAllowance implements [MOD-DE-MSG-5-5]: one PeriodicAllowance over
// the live feegrant records (union of msg_types, sum of fee_spend_limit), their
// window ends queued, the spent budget carried over. None left: revoke.
func (k Keeper) recomputeFeeAllowance(ctx context.Context, vsoa types.VSOperatorAuthorization) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read params: %w", err)
	}

	total := sdk.NewCoins()
	seen := make(map[string]bool)
	feegrantMsgTypes := make([]string, 0)
	for _, r := range vsoa.Records {
		if !r.WithFeegrant {
			continue
		}
		pv, found := k.participantKeeper().ViewParticipant(ctx, r.ParticipantId)
		if !found || (!pv.Active && !pv.Future) {
			continue
		}
		if pv.EffectiveUntil != nil {
			if err := k.WindowEndQueue.Set(ctx, collections.Join(*pv.EffectiveUntil, r.ParticipantId)); err != nil {
				return fmt.Errorf("failed to schedule window end: %w", err)
			}
		}
		for _, mt := range r.MsgTypes {
			if !seen[mt] {
				seen[mt] = true
				feegrantMsgTypes = append(feegrantMsgTypes, mt)
			}
		}
		total = total.Add(r.FeeSpendLimit...)
	}

	if len(feegrantMsgTypes) == 0 {
		return k.RevokeFeeAllowance(ctx, vsoa.CorporationId, vsoa.VsOperator)
	}
	period := params.VsOperatorFeePeriod
	reset, remaining := k.carriedFeeBudget(ctx, vsoa.CorporationId, vsoa.VsOperator, total, period)
	return k.grantFeeAllowance(ctx, vsoa.CorporationId, vsoa.VsOperator, feegrantMsgTypes, &reset, total, &period, remaining)
}
