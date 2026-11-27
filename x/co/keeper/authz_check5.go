package keeper

import (
	"context"
	stderrors "errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/co/types"
)

// ResolveCorporationByPolicyAddress implements AUTHZ-CHECK-5 for the message's
// corporation account, then runs GF-CHECK-1 on the resolved corporation at
// block time. It is the one resolver every message that carries a corporation
// account goes through.
func (k Keeper) ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (types.Corporation, error) {
	co, err := k.GetCorporationByPolicyAddress(ctx, policyAddress)
	if err != nil {
		return types.Corporation{}, err
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if err := k.gfKeeper.CheckNetworkGFAcceptance(ctx, gfCorporationView(&co), now); err != nil {
		return types.Corporation{}, err
	}
	return co, nil
}

// ResolveSigner runs AUTHZ-CHECK-5 for the corporation of a message signed by
// signer, and reports the group-proposal path: signer equal to corporation, on
// which no delegate authorization, spend limit or fee grant applies.
func (k Keeper) ResolveSigner(ctx context.Context, corporation, signer string) (types.Corporation, bool, error) {
	co, err := k.ResolveCorporationByPolicyAddress(ctx, corporation)
	if err != nil {
		return types.Corporation{}, false, err
	}
	return co, signer == corporation, nil
}

// GetCorporationByPolicyAddress returns the corporation bound to policyAddress
// without GF-CHECK-1, for lookups of an account that is not the message's
// corporation.
func (k Keeper) GetCorporationByPolicyAddress(ctx context.Context, policyAddress string) (types.Corporation, error) {
	id, err := k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if stderrors.Is(err, collections.ErrNotFound) {
		return types.Corporation{}, errors.Wrapf(types.ErrCorporationNotRegistered,
			"signing account %s has not been registered as the policy_address of a Corporation (see MOD-CO-MSG-1)", policyAddress)
	}
	if err != nil {
		return types.Corporation{}, err
	}
	return k.Corporation.Get(ctx, id)
}
