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

// authorize runs AUTHZ-CHECK-5 (with GF-CHECK-1), then AUTHZ-CHECK-1 for the
// delegate, and returns the resolved corporation id.
func (ms msgServer) authorize(ctx sdk.Context, corporation, delegate string, msgType detypes.MsgType) (uint64, error) {
	if ms.delegationKeeper == nil {
		return 0, fmt.Errorf("delegation keeper is required for delegate authorization")
	}
	corporationID, err := ms.corpIDFromAccount(ctx, corporation)
	if err != nil {
		return 0, err
	}
	if err := ms.delegationKeeper.CheckDelegateAuthorization(ctx, corporationID, delegate, msgType, ctx.BlockTime()); err != nil {
		return 0, fmt.Errorf("authorization check failed: %w", err)
	}
	return corporationID, nil
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
