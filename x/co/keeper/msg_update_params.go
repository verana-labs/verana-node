package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"github.com/verana-labs/verana-node/x/co/types"
)

// [MOD-CO-MSG-3] Update Module Parameters: the module owns no parameter, so any key aborts.
func (ms msgServer) UpdateParams(_ context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if ms.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(types.ErrInvalidSigner,
			"invalid authority; expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}
	return nil, types.ErrNoParams
}
