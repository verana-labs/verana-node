package keeper

import (
	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana-node/x/gf/types"
)

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

var _ types.MsgServer = msgServer{}

// checkNetworkPath checks the council signer of the network path. The node
// has no network governance framework to act on yet.
func (ms msgServer) checkNetworkPath(signer string) error {
	if signer != ms.GetAuthority() {
		return errors.Wrapf(sdkerrors.ErrUnauthorized, "expected %s got %s", ms.GetAuthority(), signer)
	}
	return types.ErrNetworkPathUnavailable
}
