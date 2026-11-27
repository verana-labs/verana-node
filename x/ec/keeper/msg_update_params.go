package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/verana-labs/verana-node/x/ec/types"
)

// [MOD-ES-MSG-4] Update Module Parameters: the module owns no parameter, so any key aborts.
func (ms msgServer) UpdateParams(_ context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.GetAuthority() != req.Signer {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", ms.GetAuthority(), req.Signer)
	}
	return nil, types.ErrNoParams
}
