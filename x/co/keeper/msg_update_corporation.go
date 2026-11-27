package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/co/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// UpdateCorporation implements MOD-CO-MSG-2.
//
// Rotates the DID of an existing Corporation. The corporation field on the msg
// is the policy_address (= AUTHZ-CHECK-5 subject); a delegate signer must hold
// a delegation from that Corporation for THIS msg type. Language is immutable.
func (ms msgServer) UpdateCorporation(goCtx context.Context, msg *types.MsgUpdateCorporation) (*types.MsgUpdateCorporationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// AUTHZ-CHECK-5 (with GF-CHECK-1), then AUTHZ-CHECK-1 off the group path.
	co, group, err := ms.ResolveSigner(ctx, msg.Corporation, msg.Signer)
	if err != nil {
		return nil, err
	}
	if !group {
		if err := ms.delegationKeeper.CheckDelegateAuthorization(ctx, co.Id, msg.Signer, detypes.MsgType_MSG_TYPE_UPDATE_CORPORATION, ctx.BlockTime()); err != nil {
			return nil, err
		}
	}

	if msg.Did == co.Did {
		// No-op: nothing to mutate, no event.
		return &types.MsgUpdateCorporationResponse{}, nil
	}

	// DID uniqueness gate: the new did must not be bound to a different Corporation.
	if has, err := ms.CorporationByDID.Has(ctx, msg.Did); err != nil {
		return nil, fmt.Errorf("check did uniqueness: %w", err)
	} else if has {
		return nil, errors.Wrap(types.ErrDIDAlreadyExists, msg.Did)
	}

	// [MOD-CO-MSG-2-2-1] DID ownership invariant: the corp's own ec/pp entries
	// may hold this did; entries of a different corporation must not.
	if err := ms.assertNoForeignDIDOwner(ctx, msg.Did, co.Id); err != nil {
		return nil, err
	}

	// Swap the DID index (remove old, add new) and update the Corporation row.
	if err := ms.CorporationByDID.Remove(ctx, co.Did); err != nil {
		return nil, fmt.Errorf("remove old did index: %w", err)
	}
	if err := ms.CorporationByDID.Set(ctx, msg.Did, co.Id); err != nil {
		return nil, fmt.Errorf("persist new did index: %w", err)
	}
	co.Did = msg.Did
	co.Modified = ctx.BlockTime()
	if err := ms.Corporation.Set(ctx, co.Id, co); err != nil {
		return nil, fmt.Errorf("persist corporation: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateCorporation,
		sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
		sdk.NewAttribute(types.AttributeKeyPolicyAddress, co.PolicyAddress),
		sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
	))

	return &types.MsgUpdateCorporationResponse{}, nil
}
