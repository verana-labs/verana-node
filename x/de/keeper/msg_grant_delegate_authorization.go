package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantDelegateAuthorization implements [MOD-DE-MSG-3].
func (ms msgServer) GrantDelegateAuthorization(goCtx context.Context, msg *types.MsgGrantDelegateAuthorization) (*types.MsgGrantDelegateAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-3-2] Basic checks (stateful).

	// [AUTHZ-CHECK-5] Signing corporation account MUST be a registered Corporation;
	// resolve it to co.id.
	co, err := ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}

	// Group-proposal path: delegate equals corporation and AUTHZ-CHECK-5 alone
	// applies. Otherwise AUTHZ-CHECK MUST pass for the delegate.
	if msg.Delegate != msg.Corporation {
		if err := ms.CheckDelegateAuthorization(
			ctx,
			co.Id,
			msg.Delegate,
			types.MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION,
			now,
		); err != nil {
			return nil, err
		}
		// [MOD-DE-MSG-3] A delegate cannot grant itself new msg_types (escalation).
		if msg.Grantee == msg.Delegate {
			return nil, fmt.Errorf("delegate cannot grant authorization to itself; use a group proposal")
		}
	}

	// Expiration must be in the future if specified.
	if msg.Expiration != nil && !msg.Expiration.After(now) {
		return nil, types.ErrExpirationInPast
	}

	// authz_spend_limit_period must be valid if authz_spend_limit is set.
	if len(msg.AuthzSpendLimit) > 0 && msg.AuthzSpendLimitPeriod != nil && *msg.AuthzSpendLimitPeriod <= 0 {
		return nil, fmt.Errorf("authz_spend_limit_period must be a positive duration")
	}

	// grantee MUST NOT be the corporation's own policy_address.
	if types.IsOwnPolicyAddress(co.PolicyAddress, msg.Grantee) {
		return nil, types.ErrOwnPolicyAddress
	}

	// Mutual exclusivity: a VSOperatorAuthorization MUST NOT exist for
	// (corporation_id, grantee).
	hasVSOA, err := ms.VSOAByCorpOp.Has(ctx, collections.Join(co.Id, msg.Grantee))
	if err != nil {
		return nil, fmt.Errorf("failed to check VSOperatorAuthorization: %w", err)
	}
	if hasVSOA {
		return nil, types.ErrVSOperatorAuthzExists
	}

	// [MOD-DE-MSG-3-4] Execution.

	// 1. Lookup existing DelegateAuthorization by (co.id, grantee): preserve id
	// on in-place update, allocate a fresh one otherwise.
	existing, found, err := ms.getDelegateAuthorizationByCorpDelegate(ctx, co.Id, msg.Grantee)
	if err != nil {
		return nil, err
	}
	var daID uint64
	if found {
		daID = existing.Id
	} else {
		daID, err = ms.nextDelegateAuthorizationID(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Seed runtime balances at grant time per [MOD-DE-MSG-3] / AUTHZ-CHECK-1.
	da := types.DelegateAuthorization{
		Id:             daID,
		CorporationId:  co.Id,
		Delegate:       msg.Grantee,
		MsgTypes:       msg.MsgTypes,
		SpendLimit:     msg.AuthzSpendLimit,
		RemainingSpend: msg.AuthzSpendLimit,
		Expiration:     msg.Expiration,
	}
	// period is ignored without a spend limit; storing it otherwise would make
	// the authorization auto-renew its expiration forever.
	if len(msg.AuthzSpendLimit) > 0 {
		da.Period = msg.AuthzSpendLimitPeriod
	}
	if err := ms.DelegateAuthorizations.Set(ctx, daID, da); err != nil {
		return nil, fmt.Errorf("failed to set DelegateAuthorization: %w", err)
	}
	if err := ms.DelegateAuthorizationByCorpDelegate.Set(ctx, collections.Join(co.Id, msg.Grantee), daID); err != nil {
		return nil, fmt.Errorf("failed to set DelegateAuthorization index: %w", err)
	}

	// 2. Handle fee grant.
	if !msg.WithFeegrant {
		if err := ms.RevokeFeeAllowance(ctx, co.Id, msg.Grantee); err != nil {
			return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
		}
	} else {
		// period is ignored when feegrant_spend_limit is not set (MOD-DE-MSG-3-2).
		feegrantPeriod := msg.FeegrantSpendLimitPeriod
		if len(msg.FeegrantSpendLimit) == 0 {
			feegrantPeriod = nil
		}
		if err := ms.GrantFeeAllowance(
			ctx,
			co.Id,
			msg.Grantee,
			msg.MsgTypes,
			msg.Expiration,
			msg.FeegrantSpendLimit,
			feegrantPeriod,
		); err != nil {
			return nil, fmt.Errorf("failed to grant fee allowance: %w", err)
		}
	}

	// 3. Emit events.
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeGrantDelegateAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthzID, strconv.FormatUint(daID, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(co.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
			sdk.NewAttribute(types.AttributeKeyWithFeegrant, fmt.Sprintf("%t", msg.WithFeegrant)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgGrantDelegateAuthorizationResponse{}, nil
}
