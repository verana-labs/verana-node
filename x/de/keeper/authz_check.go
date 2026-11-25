package keeper

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// emitDelegateAuthzUpdated signals that an AUTHZ-CHECK-1 path mutated the
// delegate authorization (spend debit or cycle renewal). It carries only the id;
// the indexer re-reads authoritative state via ABCI at this height.
func (k Keeper) emitDelegateAuthzUpdated(ctx context.Context, authzID uint64) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDelegateAuthorizationUpdated,
			sdk.NewAttribute(types.AttributeKeyAuthzID, strconv.FormatUint(authzID, 10)),
		),
	)
}

// CheckDelegateAuthorization implements [AUTHZ-CHECK-1] for delegate acting for
// the corporation that AUTHZ-CHECK-5 resolved to corporationID: existence,
// expiration/period renewal, msg_type membership and GF-CHECK-2. The spend
// debit (step 3) is applied by ConsumeDelegateSpend once the handler knows the
// amount.
func (k Keeper) CheckDelegateAuthorization(
	ctx context.Context,
	corporationID uint64,
	delegate string,
	msgType types.MsgType,
	now time.Time,
) error {
	_, err := k.checkDelegateAuthorizationCore(ctx, corporationID, delegate, msgType, now)
	return err
}

// ConsumeDelegateSpend implements [AUTHZ-CHECK-1] step 3: it verifies the
// delegate authorization's remaining_spend covers `amount` and debits it.
// Callers pass the operation's nominal fund commitment (fees + face-value trust
// deposit). It is a no-op when no spend_limit is configured. It reuses
// checkDelegateAuthorizationCore so the period-reset / expiry / msg_type
// invariant holds at debit time. The debit is reverted with the transaction if
// any later step in the handler fails.
func (k Keeper) ConsumeDelegateSpend(
	ctx context.Context,
	corporationID uint64,
	delegate string,
	msgType types.MsgType,
	now time.Time,
	amount sdk.Coins,
) error {
	if amount.IsZero() {
		return nil
	}
	da, err := k.checkDelegateAuthorizationCore(ctx, corporationID, delegate, msgType, now)
	if err != nil {
		return err
	}
	if len(da.SpendLimit) == 0 {
		return nil
	}
	if !da.RemainingSpend.IsAllGTE(amount) {
		return fmt.Errorf("%w: spend %s exceeds remaining %s",
			types.ErrAuthzSpendLimitExceeded, amount.String(), da.RemainingSpend.String())
	}
	da.RemainingSpend = da.RemainingSpend.Sub(amount...)
	if err := k.DelegateAuthorizations.Set(ctx, da.Id, da); err != nil {
		return fmt.Errorf("failed to debit delegate spend: %w", err)
	}
	k.emitDelegateAuthzUpdated(ctx, da.Id)
	return nil
}

// checkDelegateAuthorizationCore performs the period-renewal / expiration +
// msg_type checks and GF-CHECK-2, and returns the loaded DelegateAuthorization.
func (k Keeper) checkDelegateAuthorizationCore(
	ctx context.Context,
	corporationID uint64,
	delegate string,
	msgType types.MsgType,
	now time.Time,
) (types.DelegateAuthorization, error) {
	// 1. Load DelegateAuthorization via the (corporation_id, delegate) index.
	da, found, err := k.getDelegateAuthorizationByCorpDelegate(ctx, corporationID, delegate)
	if err != nil {
		return types.DelegateAuthorization{}, err
	}
	if !found {
		return types.DelegateAuthorization{}, types.ErrAuthzNotFound
	}

	// 2. Expiration / period auto-renewal (AUTHZ-CHECK-1 step 2).
	if da.Expiration != nil {
		if da.Period != nil && *da.Period > 0 && !da.Expiration.After(now) {
			// Period elapsed: reset the spend balance and roll expiration forward.
			if len(da.SpendLimit) > 0 {
				da.RemainingSpend = da.SpendLimit
			}
			newExp := now.Add(*da.Period)
			da.Expiration = &newExp
			if err := k.DelegateAuthorizations.Set(ctx, da.Id, da); err != nil {
				return types.DelegateAuthorization{}, fmt.Errorf("failed to persist authz renewal: %w", err)
			}
			k.emitDelegateAuthzUpdated(ctx, da.Id)
		} else if !da.Expiration.After(now) {
			return types.DelegateAuthorization{}, types.ErrAuthzExpired
		}
	}

	// 3. Check that the requested msg type is authorized.
	if !slices.Contains(da.MsgTypes, msgType) {
		return types.DelegateAuthorization{}, fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgType)
	}

	// 4. GF-CHECK-2.
	if err := k.gfRef.K.CheckDelegateCGFAcceptance(ctx, corporationID, &da, now); err != nil {
		return types.DelegateAuthorization{}, err
	}

	return da, nil
}
