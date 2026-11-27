package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// RevokeDelegateAuthorization implements [MOD-DE-MSG-4].
func (ms msgServer) RevokeDelegateAuthorization(goCtx context.Context, msg *types.MsgRevokeDelegateAuthorization) (*types.MsgRevokeDelegateAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-4-2] Basic checks (stateful).

	// [AUTHZ-CHECK-5] Corporation account MUST be a registered Corporation;
	// resolve it to co.id. AUTHZ-CHECK applies off the group-proposal path.
	co, group, err := ms.corporationKeeper().ResolveSigner(ctx, msg.Corporation, msg.Signer)
	if err != nil {
		return nil, err
	}
	if !group {
		if err := ms.CheckDelegateAuthorization(
			ctx,
			co.Id,
			msg.Signer,
			types.MsgType_MSG_TYPE_REVOKE_DELEGATE_AUTHORIZATION,
			now,
		); err != nil {
			return nil, err
		}
	}

	// A DelegateAuthorization MUST exist for (co.id, grantee).
	existing, found, err := ms.getDelegateAuthorizationByCorpDelegate(ctx, co.Id, msg.Grantee)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrDelegateAuthzNotFound
	}

	// [MOD-DE-MSG-4-4] Execution.

	// 1. Delete the DelegateAuthorization and its secondary index.
	if err := ms.DelegateAuthorizations.Remove(ctx, existing.Id); err != nil {
		return nil, fmt.Errorf("failed to remove DelegateAuthorization: %w", err)
	}
	if err := ms.DelegateAuthorizationByCorpDelegate.Remove(ctx, collections.Join(co.Id, msg.Grantee)); err != nil {
		return nil, fmt.Errorf("failed to remove DelegateAuthorization index: %w", err)
	}

	// 2. Revoke Fee Allowance for (co.id, grantee).
	if err := ms.RevokeFeeAllowance(ctx, co.Id, msg.Grantee); err != nil {
		return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
	}

	// 3. Emit events.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeDelegateAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthzID, strconv.FormatUint(existing.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(co.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return &types.MsgRevokeDelegateAuthorizationResponse{}, nil
}
