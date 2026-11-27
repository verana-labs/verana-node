package keeper

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantDelegateAuthorization implements [MOD-DE-MSG-3].
func (ms msgServer) GrantDelegateAuthorization(goCtx context.Context, msg *types.MsgGrantDelegateAuthorization) (*types.MsgGrantDelegateAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-3-2] Basic checks (stateful).

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
			types.MsgType_MSG_TYPE_GRANT_DELEGATE_AUTHORIZATION,
			now,
		); err != nil {
			return nil, err
		}
	}

	// Expiration must be in the future if specified.
	if msg.Expiration != nil && !msg.Expiration.After(now) {
		return nil, types.ErrExpirationInPast
	}

	// grantee MUST NOT be the corporation's own policy_address.
	if types.IsOwnPolicyAddress(co.PolicyAddress, msg.Grantee) {
		return nil, types.ErrOwnPolicyAddress
	}

	grantee, err := sdk.AccAddressFromBech32(msg.Grantee)
	if err != nil {
		return nil, fmt.Errorf("invalid grantee address: %w", err)
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, fmt.Errorf("invalid signer address: %w", err)
	}
	// A delegate does not grant an authorization to itself.
	if grantee.Equals(signer) {
		return nil, fmt.Errorf("grantee must not be the signer")
	}

	// Mutual exclusivity: grantee MUST NOT be a VS account of the corporation,
	// whose index keeps the account as spelled, in either bech32 case.
	for _, spelling := range []string{grantee.String(), strings.ToUpper(grantee.String())} {
		hasVSOA, err := ms.VSOAByCorpOp.Has(ctx, collections.Join(co.Id, spelling))
		if err != nil {
			return nil, fmt.Errorf("failed to check VSOperatorAuthorization: %w", err)
		}
		if hasVSOA {
			return nil, types.ErrVSOperatorAuthzExists
		}
	}

	// grantee MUST NOT be the policy_address of a corporation co sponsors: the
	// sponsorship holds that x/feegrant slot.
	sponsored, found, err := ms.corporationKeeper().GetCorporationByPolicyAddress(ctx, grantee.String())
	if err != nil {
		return nil, err
	}
	if found {
		sponsoring, err := ms.Sponsorships.Has(ctx, collections.Join(co.Id, sponsored.Id))
		if err != nil {
			return nil, err
		}
		if sponsoring {
			return nil, errorsmod.Wrapf(types.ErrSponsorshipSlotTaken, "grantee is the policy_address of sponsored corporation %d", sponsored.Id)
		}
	}

	// [MOD-DE-MSG-3-4] Execution: create, or update keeping accepted_cgf_version.
	existing, _, err := ms.getDelegateAuthorization(ctx, co.Id, grantee)
	if err != nil {
		return nil, err
	}
	da := types.DelegateAuthorization{
		CorporationId:      co.Id,
		Delegate:           grantee.String(),
		MsgTypes:           msg.MsgTypes,
		SpendLimit:         msg.AuthzSpendLimit,
		RemainingSpend:     msg.AuthzSpendLimit,
		Expiration:         msg.Expiration,
		AcceptedCgfVersion: existing.AcceptedCgfVersion,
	}
	if len(msg.AuthzSpendLimit) > 0 {
		da.Period = msg.AuthzSpendLimitPeriod
	}
	if err := ms.setDelegateAuthorization(ctx, da); err != nil {
		return nil, fmt.Errorf("failed to set DelegateAuthorization: %w", err)
	}

	if len(msg.FeegrantSpendLimit) == 0 {
		if err := ms.RevokeFeeAllowance(ctx, co.Id, da.Delegate); err != nil {
			return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
		}
	} else {
		// The allowance also pays the delegate's CGF acceptance, once that message exists.
		feeMsgTypes := msg.MsgTypes
		if cgf := types.MsgType_MSG_TYPE_ACCEPT_CORPORATION_GOVERNANCE_FRAMEWORK; len(cgf.URLs()) > 0 {
			feeMsgTypes = append(slices.Clone(msg.MsgTypes), cgf)
		}
		if err := ms.GrantFeeAllowance(
			ctx,
			co.Id,
			da.Delegate,
			feeMsgTypes,
			msg.Expiration,
			msg.FeegrantSpendLimit,
			msg.FeegrantSpendLimitPeriod,
		); err != nil {
			return nil, fmt.Errorf("failed to grant fee allowance: %w", err)
		}
	}

	if err := ms.emitDelegateAuthorizationEvent(ctx, types.EventTypeGrantDelegateAuthorization, co.Id, grantee, msg); err != nil {
		return nil, err
	}
	return &types.MsgGrantDelegateAuthorizationResponse{}, nil
}

// emitDelegateAuthorizationEvent emits the event of MOD-DE-MSG-3 or MOD-DE-MSG-4.
// It always carries the message: the delegate may itself be a group policy
// account executing through a proposal.
func (k Keeper) emitDelegateAuthorizationEvent(ctx sdk.Context, eventType string, corporationID uint64, delegate sdk.AccAddress, msg proto.Message) error {
	ev := events.New(types.ModuleName, eventType).
		Uint(events.AttrCorporationID, corporationID).
		Addr(types.AttributeKeyDelegate, delegate)
	if err := ev.Msg(k.cdc, msg); err != nil {
		return err
	}
	ev.Emit(ctx)
	return nil
}
