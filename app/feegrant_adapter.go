package app

import (
	"context"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
)

// deFeegrantAdapter adapts the x/feegrant keeper to MOD-DE's FeegrantKeeper.
type deFeegrantAdapter struct {
	feegrantkeeper.Keeper
}

func newDeFeegrantAdapter(k feegrantkeeper.Keeper) deFeegrantAdapter {
	return deFeegrantAdapter{Keeper: k}
}

// RevokeAllowance mirrors the keeper's unexported revokeAllowance. The msg
// server rejects granter == grantee, which GrantAllowance accepts.
func (a deFeegrantAdapter) RevokeAllowance(ctx context.Context, granter, grantee sdk.AccAddress) error {
	grant, err := a.GetAllowance(ctx, granter, grantee)
	if err != nil {
		return err
	}
	if err := a.FeeAllowance.Remove(ctx, collections.Join(grantee, granter)); err != nil {
		return err
	}
	exp, err := grant.ExpiresAt()
	if err != nil {
		return err
	}
	if exp != nil {
		if err := a.FeeAllowanceQueue.Remove(ctx, collections.Join3(*exp, grantee, granter)); err != nil {
			return err
		}
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			feegrant.EventTypeRevokeFeeGrant,
			sdk.NewAttribute(feegrant.AttributeKeyGranter, granter.String()),
			sdk.NewAttribute(feegrant.AttributeKeyGrantee, grantee.String()),
		),
	)
	return nil
}
