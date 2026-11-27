package keeper

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/di/types"
)

// StoreDigest implements [MOD-DI-MSG-1] Store Digest.
func (ms msgServer) StoreDigest(goCtx context.Context, msg *types.MsgStoreDigest) (*types.MsgStoreDigestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DI-MSG-1-2-1] digest validation, matching the module-call path.
	if err := types.ValidateDigestString(msg.Digest); err != nil {
		return nil, err
	}

	// [AUTHZ-CHECK-5] (with GF-CHECK-1): AUTHZ-CHECK-1 consumes the co it resolves.
	co, group, err := ms.coKeeper.ResolveSigner(ctx, msg.Corporation, msg.Signer)
	if err != nil {
		return nil, err
	}

	// [MOD-DI-MSG-1-2-1] [AUTHZ-CHECK-1] Verify delegate authorization.
	if !group {
		if err := ms.delegationKeeper.CheckDelegateAuthorization(
			ctx,
			co.Id,
			msg.Signer,
			detypes.MsgType_MSG_TYPE_STORE_DIGEST,
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-DI-MSG-1-3] A duplicate digest is an idempotent no-op (returns success).
	if has, err := ms.Keeper.Digests.Has(ctx, msg.Digest); err != nil {
		return nil, err
	} else if has {
		return &types.MsgStoreDigestResponse{}, nil
	}

	// [MOD-DI-MSG-1-3] Execution — Create Digest record
	digest := types.Digest{
		Digest:  msg.Digest,
		Created: now,
	}

	if err := ms.Digests.Set(ctx, msg.Digest, digest); err != nil {
		return nil, fmt.Errorf("failed to store digest: %w", err)
	}

	emitStoreDigestEvent(ctx, msg.Corporation, msg.Signer, msg.Digest, types.AttributeValueSourceMsg, now)

	return &types.MsgStoreDigestResponse{}, nil
}

// StoreDigestModuleCall is the module-call entry point for [MOD-DI-MSG-1].
// It can be called directly by the perm module (CreateOrUpdatePermissionSession)
// with no signer/AUTHZ checks. It applies the same digest validation as the Msg
// path [DI-MIN-2].
func (k Keeper) StoreDigestModuleCall(ctx context.Context, authority, digest string) error {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if err := types.ValidateDigestString(digest); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// A duplicate digest is an idempotent no-op.
	if has, err := k.Digests.Has(ctx, digest); err != nil {
		return err
	} else if has {
		return nil
	}

	if err := k.Digests.Set(ctx, digest, types.Digest{Digest: digest, Created: now}); err != nil {
		return fmt.Errorf("failed to store digest: %w", err)
	}

	emitStoreDigestEvent(sdkCtx, authority, "", digest, types.AttributeValueSourceModuleCall, now)

	return nil
}

func emitStoreDigestEvent(ctx sdk.Context, corporation, signer, digest, source string, now time.Time) {
	attrs := []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeyCorporation, corporation),
		sdk.NewAttribute(types.AttributeKeyDigest, digest),
		sdk.NewAttribute(types.AttributeKeySource, source),
		sdk.NewAttribute(types.AttributeKeyTimestamp, now.UTC().Format(time.RFC3339Nano)),
	}
	if signer != "" {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeySigner, signer))
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeStoreDigest, attrs...))
}
