package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// emitOperatorAuthzUpdated signals that an AUTHZ-CHECK-1 path mutated the
// operator authorization (spend debit or cycle renewal). It carries only the id;
// the indexer re-reads authoritative state via ABCI at this height.
func (k Keeper) emitOperatorAuthzUpdated(ctx context.Context, authzID uint64) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOperatorAuthorizationUpdated,
			sdk.NewAttribute(types.AttributeKeyAuthzID, strconv.FormatUint(authzID, 10)),
		),
	)
}

// CheckOperatorAuthorization implements [AUTHZ-CHECK-1] membership: existence,
// expiration/period renewal, and msg_type membership. The spend debit (step 3)
// is applied by ConsumeOperatorSpend once the handler knows the amount.
//
// `corporation` is the signing corporation account (policy_address); it is
// resolved to co.id via AUTHZ-CHECK-5 before the (corporation_id, operator)
// index lookup.
func (k Keeper) CheckOperatorAuthorization(
	ctx context.Context,
	corporation string,
	operator string,
	msgTypeURL string,
	now time.Time,
) error {
	_, err := k.checkOperatorAuthorizationCore(ctx, corporation, operator, msgTypeURL, now)
	return err
}

// ConsumeOperatorSpend implements [AUTHZ-CHECK-1] step 3: it verifies the
// operator authorization's remaining_spend covers `amount` and debits it.
// Callers pass the operation's nominal fund commitment (fees + face-value trust
// deposit). It is a no-op when operator=="" (corporation acting alone via a group
// proposal) or no spend_limit is configured. It reuses
// checkOperatorAuthorizationCore so the period-reset / expiry / msg_type
// invariant holds at debit time. The debit is reverted with the transaction if
// any later step in the handler fails.
func (k Keeper) ConsumeOperatorSpend(
	ctx context.Context,
	corporation string,
	operator string,
	msgTypeURL string,
	now time.Time,
	amount sdk.Coins,
) error {
	if operator == "" || amount.IsZero() {
		return nil
	}
	oa, err := k.checkOperatorAuthorizationCore(ctx, corporation, operator, msgTypeURL, now)
	if err != nil {
		return err
	}
	if len(oa.SpendLimit) == 0 {
		return nil
	}
	if !oa.RemainingSpend.IsAllGTE(amount) {
		return fmt.Errorf("%w: spend %s exceeds remaining %s",
			types.ErrAuthzSpendLimitExceeded, amount.String(), oa.RemainingSpend.String())
	}
	oa.RemainingSpend = oa.RemainingSpend.Sub(amount...)
	if err := k.OperatorAuthorizations.Set(ctx, oa.Id, oa); err != nil {
		return fmt.Errorf("failed to debit operator spend: %w", err)
	}
	k.emitOperatorAuthzUpdated(ctx, oa.Id)
	return nil
}

// checkOperatorAuthorizationCore performs the period-renewal / expiration +
// msg_type checks and returns the loaded OperatorAuthorization.
func (k Keeper) checkOperatorAuthorizationCore(
	ctx context.Context,
	corporation string,
	operator string,
	msgTypeURL string,
	now time.Time,
) (types.OperatorAuthorization, error) {
	// If operator is empty, the corporation is acting alone (group proposal) — skip.
	if operator == "" {
		return types.OperatorAuthorization{}, nil
	}

	// Resolve the signing corporation account to its co.id (AUTHZ-CHECK-5). An
	// unregistered corporation cannot have granted any authorization.
	co, err := k.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, corporation)
	if err != nil {
		return types.OperatorAuthorization{}, types.ErrAuthzNotFound
	}

	// 1. Load OperatorAuthorization via the (corporation_id, operator) index.
	oa, found, err := k.getOperatorAuthorizationByCorpOp(ctx, co.Id, operator)
	if err != nil {
		return types.OperatorAuthorization{}, err
	}
	if !found {
		return types.OperatorAuthorization{}, types.ErrAuthzNotFound
	}

	// 2. Expiration / period auto-renewal (AUTHZ-CHECK-1 step 2).
	if oa.Expiration != nil {
		if oa.Period != nil && *oa.Period > 0 && !oa.Expiration.After(now) {
			// Period elapsed: reset the spend balance and roll expiration forward.
			if len(oa.SpendLimit) > 0 {
				oa.RemainingSpend = oa.SpendLimit
			}
			newExp := now.Add(*oa.Period)
			oa.Expiration = &newExp
			if err := k.OperatorAuthorizations.Set(ctx, oa.Id, oa); err != nil {
				return types.OperatorAuthorization{}, fmt.Errorf("failed to persist authz renewal: %w", err)
			}
			k.emitOperatorAuthzUpdated(ctx, oa.Id)
		} else if !oa.Expiration.After(now) {
			return types.OperatorAuthorization{}, types.ErrAuthzExpired
		}
	}

	// 3. Check that the requested msg type is authorized.
	found = false
	for _, mt := range oa.MsgTypes {
		if mt == msgTypeURL {
			found = true
			break
		}
	}
	if !found {
		return types.OperatorAuthorization{}, fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgTypeURL)
	}

	return oa, nil
}
