package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// TriggerResolver implements [MOD-PP-MSG-15]. It is event-only: it does not
// modify VPR state. It emits EventTypeTriggerResolver so off-chain trust
// resolvers (e.g. the verana-indexer) re-resolve the participant's did.
func (ms msgServer) TriggerResolver(goCtx context.Context, msg *types.MsgTriggerResolver) (*types.MsgTriggerResolverResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for delegate authorization")
	}

	// [AUTHZ-CHECK-5]
	co, group, err := ms.coKeeper.ResolveSigner(ctx, msg.Corporation, msg.Signer)
	if err != nil {
		return nil, err
	}

	// [MOD-PP-MSG-15-2-1] Basic checks.
	perm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("participant %d not found", msg.Id)
	}
	if err := IsValidParticipant(perm, now); err != nil {
		return nil, fmt.Errorf("participant %d is not active: %w", msg.Id, err)
	}

	// [MOD-PP-MSG-15-2-2] Authorization (at least one path MUST pass).
	if err := ms.authorizeTriggerResolver(ctx, msg.Corporation, co.Id, msg.Signer, group, perm, now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PP-MSG-15-3] Execution: no state mutation, emit the event.
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTriggerResolver,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeySigner, msg.Signer),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgTriggerResolverResponse{}, nil
}

// authorizeTriggerResolver runs the two-path [MOD-PP-MSG-15-2-2] authorization.
// corpAccount is the message's corporation account, corpID its resolved co.id,
// group whether signer is on the group-proposal path, perm the target
// Participant.
//
// Path 1 enforces AUTHZ-CHECK-3 plus the with_feegrant gate (AUTHZ-CHECK-4);
// Path 2 accepts the group-proposal path, else enforces AUTHZ-CHECK-1.
func (ms msgServer) authorizeTriggerResolver(
	ctx sdk.Context, corpAccount string, corpID uint64, signer string, group bool, perm types.Participant, now time.Time,
) error {
	const msgType = detypes.MsgType_MSG_TYPE_TRIGGER_RESOLVER

	// Path 1 — vs_operator of the target participant (AUTHZ-CHECK-3).
	if corpID == perm.CorporationId && signer == perm.VsOperator {
		if err := ms.delegationKeeper.CheckVSOperatorAuthorizationOnParticipant(
			ctx, corpID, signer, perm.Id, msgType); err == nil {
			// [AUTHZ-CHECK-4] with_feegrant gate when the corporation pays the tx fee.
			if err := ms.checkVSOperatorFeeGrant(ctx, perm.Id, corpAccount); err != nil {
				return err
			}
			return nil
		}
	}

	// Path 2 — ancestor validator walk; the target perm itself is excluded
	// (AUTHZ-CHECK-1 for each active, same-corp ancestor).
	visited := map[uint64]bool{perm.Id: true}
	v := perm
	for v.ValidatorParticipantId != 0 && v.ValidatorParticipantId != perm.Id {
		next := v.ValidatorParticipantId
		if visited[next] {
			// Defensive: the participant tree is acyclic, but never loop forever.
			break
		}
		visited[next] = true

		parent, err := ms.Keeper.GetParticipantByID(ctx, next)
		if err != nil {
			break
		}
		v = parent

		if err := IsValidParticipant(v, now); err != nil {
			continue // skip inactive ancestor
		}
		if corpID != v.CorporationId {
			continue
		}
		if group {
			return nil
		}
		if err := ms.delegationKeeper.CheckDelegateAuthorization(
			ctx, corpID, signer, msgType, now); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no authorization path matched for signer %s on participant %d", signer, perm.Id)
}
