package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantOperatorAuthorization implements [MOD-DE-MSG-3].
func (ms msgServer) GrantOperatorAuthorization(goCtx context.Context, msg *types.MsgGrantOperatorAuthorization) (*types.MsgGrantOperatorAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-3-2] Basic checks (stateful).

	// [AUTHZ-CHECK-5] Signing corporation account MUST be a registered Corporation;
	// resolve it to co.id.
	co, err := ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}

	// [AUTHZ-CHECK-1] The operator is the signer. When it equals the corporation
	// policy_address the corporation is acting alone (group proposal) and the
	// check is skipped; otherwise the operator's delegation MUST cover this msg.
	if msg.Operator != msg.Corporation {
		if err := ms.CheckOperatorAuthorization(
			ctx,
			msg.Corporation,
			msg.Operator,
			"/verana.de.v1.MsgGrantOperatorAuthorization",
			now,
		); err != nil {
			return nil, err
		}
		// [MOD-DE-MSG-3] An operator cannot grant itself new msg_types (escalation).
		if msg.Grantee == msg.Operator {
			return nil, fmt.Errorf("operator cannot grant authorization to itself; use a group proposal")
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

	// 1. Lookup existing OperatorAuthorization by (co.id, grantee): preserve id
	// on in-place update, allocate a fresh one otherwise.
	existing, found, err := ms.getOperatorAuthorizationByCorpOp(ctx, co.Id, msg.Grantee)
	if err != nil {
		return nil, err
	}
	var oaID uint64
	if found {
		oaID = existing.Id
	} else {
		oaID, err = ms.nextOperatorAuthorizationID(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Seed runtime balances at grant time per [MOD-DE-MSG-3] / AUTHZ-CHECK-1.
	oa := types.OperatorAuthorization{
		Id:             oaID,
		CorporationId:  co.Id,
		Operator:       msg.Grantee,
		MsgTypes:       msg.MsgTypes,
		SpendLimit:     msg.AuthzSpendLimit,
		RemainingSpend: msg.AuthzSpendLimit,
		Expiration:     msg.Expiration,
	}
	// period is ignored without a spend limit; storing it otherwise would make
	// the authorization auto-renew its expiration forever.
	if len(msg.AuthzSpendLimit) > 0 {
		oa.Period = msg.AuthzSpendLimitPeriod
	}
	if err := ms.OperatorAuthorizations.Set(ctx, oaID, oa); err != nil {
		return nil, fmt.Errorf("failed to set OperatorAuthorization: %w", err)
	}
	if err := ms.OperatorAuthorizationByCorpOp.Set(ctx, collections.Join(co.Id, msg.Grantee), oaID); err != nil {
		return nil, fmt.Errorf("failed to set OperatorAuthorization index: %w", err)
	}

	// 2. Handle fee grant.
	if !msg.WithFeegrant {
		if err := ms.RevokeFeeAllowance(ctx, co.Id, msg.Grantee); err != nil {
			return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
		}
	} else {
		if err := ms.GrantFeeAllowance(
			ctx,
			co.Id,
			msg.Grantee,
			msg.MsgTypes,
			msg.Expiration,
			msg.FeegrantSpendLimit,
			msg.FeegrantSpendLimitPeriod,
		); err != nil {
			return nil, fmt.Errorf("failed to grant fee allowance: %w", err)
		}
	}

	// 3. Emit events.
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeGrantOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthzID, strconv.FormatUint(oaID, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(co.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
			sdk.NewAttribute(types.AttributeKeyWithFeegrant, fmt.Sprintf("%t", msg.WithFeegrant)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgGrantOperatorAuthorizationResponse{}, nil
}
