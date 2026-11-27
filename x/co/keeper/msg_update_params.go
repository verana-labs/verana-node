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
	if ms.GetAuthority() != msg.Signer {
		return nil, errors.Wrapf(types.ErrInvalidSigner,
			"invalid signer; expected %s, got %s", ms.GetAuthority(), msg.Signer)
	}
	return nil, types.ErrNoParams
}
