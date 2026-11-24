package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// RevokeDelegateAuthorization implements [MOD-DE-MSG-4].
func (ms msgServer) RevokeDelegateAuthorization(goCtx context.Context, msg *types.MsgRevokeDelegateAuthorization) (*types.MsgRevokeDelegateAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-4-2] Basic checks (stateful).

	// [AUTHZ-CHECK-5] Signing corporation account MUST be a registered Corporation;
	// resolve it to co.id.
	co, err := ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}

	// Group-proposal path: delegate is the corporation's policy account and
	// AUTHZ-CHECK-5 alone applies. Otherwise AUTHZ-CHECK MUST pass for the delegate.
	if !types.IsOwnPolicyAddress(co.PolicyAddress, msg.Delegate) {
		if err := ms.CheckDelegateAuthorization(
			ctx,
			co.Id,
			msg.Delegate,
			types.MsgType_MSG_TYPE_REVOKE_DELEGATE_AUTHORIZATION,
			now,
		); err != nil {
			return nil, err
		}
	}

	// A DelegateAuthorization MUST exist for (co.id, grantee).
	grantee, err := sdk.AccAddressFromBech32(msg.Grantee)
	if err != nil {
		return nil, fmt.Errorf("invalid grantee address: %w", err)
	}
	key := collections.Join(co.Id, grantee)
	has, err := ms.DelegateAuthorizations.Has(ctx, key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, types.ErrDelegateAuthzNotFound
	}

	// [MOD-DE-MSG-4-4] Execution.
	if err := ms.DelegateAuthorizations.Remove(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to remove DelegateAuthorization: %w", err)
	}
	if err := ms.RevokeFeeAllowance(ctx, co.Id, grantee.String()); err != nil {
		return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
	}

	if err := ms.emitDelegateAuthorizationEvent(ctx, types.EventTypeRevokeDelegateAuthorization, co.Id, grantee, msg); err != nil {
		return nil, err
	}
	return &types.MsgRevokeDelegateAuthorizationResponse{}, nil
}
