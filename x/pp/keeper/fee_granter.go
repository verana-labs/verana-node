package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type feeGranterKey struct{}

// NewFeeGranterAnteHandler wraps next and stores the tx fee granter in the
// context, so [AUTHZ-CHECK-4] reads it once per tx instead of once per message.
func NewFeeGranterAnteHandler(next sdk.AnteHandler) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		if feeTx, ok := tx.(sdk.FeeTx); ok {
			ctx = ctx.WithValue(feeGranterKey{}, feeTx.FeeGranter())
		}
		return next(ctx, tx, simulate)
	}
}

func feeGranterFromContext(ctx sdk.Context) []byte {
	granter, _ := ctx.Value(feeGranterKey{}).([]byte)
	return granter
}
