package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// corpIDFromAccount resolves the message's corporation account (policy_address)
// to its uint64 Corporation id through AUTHZ-CHECK-5, which also runs GF-CHECK-1.
// The Participant entity persists corporation_id; callers pass the account from
// the Msg.
func (ms msgServer) corpIDFromAccount(ctx context.Context, account string) (uint64, error) {
	co, err := ms.coKeeper.ResolveCorporationByPolicyAddress(ctx, account)
	if err != nil {
		return 0, err
	}
	return co.Id, nil
}

// authorize runs AUTHZ-CHECK-5 (with GF-CHECK-1), then AUTHZ-CHECK-1 for a
// delegate signer, and returns the resolved corporation id and whether the
// signer is on the group-proposal path.
func (ms msgServer) authorize(ctx sdk.Context, corporation, signer string, msgType detypes.MsgType) (uint64, bool, error) {
	if ms.delegationKeeper == nil {
		return 0, false, fmt.Errorf("delegation keeper is required for delegate authorization")
	}
	co, group, err := ms.coKeeper.ResolveSigner(ctx, corporation, signer)
	if err != nil || group {
		return co.Id, group, err
	}
	if err := ms.delegationKeeper.CheckDelegateAuthorization(ctx, co.Id, signer, msgType, ctx.BlockTime()); err != nil {
		return 0, false, fmt.Errorf("authorization check failed: %w", err)
	}
	return co.Id, false, nil
}

// corpAccountFromID resolves a Participant's corporation_id back to the
// Corporation policy_address account used for fund-flows (trust deposit,
// feegrant, slashing).
func (ms msgServer) corpAccountFromID(ctx context.Context, id uint64) (string, error) {
	co, ok := ms.coKeeper.ResolveByID(ctx, id)
	if !ok {
		return "", fmt.Errorf("corporation %d not found", id)
	}
	return co.PolicyAddress, nil
}
