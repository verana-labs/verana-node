package app

import (
	"context"

	feegrant "cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// deFeegrantAdapter adapts the x/feegrant keeper to MOD-DE's FeegrantKeeper; revoke routes via the msg server.
type deFeegrantAdapter struct {
	feegrantkeeper.Keeper
	msgServer feegrant.MsgServer
}

func newDeFeegrantAdapter(k feegrantkeeper.Keeper) deFeegrantAdapter {
	return deFeegrantAdapter{Keeper: k, msgServer: feegrantkeeper.NewMsgServerImpl(k)}
}

func (a deFeegrantAdapter) RevokeAllowance(ctx context.Context, granter, grantee sdk.AccAddress) error {
	_, err := a.msgServer.RevokeAllowance(ctx, &feegrant.MsgRevokeAllowance{
		Granter: granter.String(),
		Grantee: grantee.String(),
	})
	return err
}
