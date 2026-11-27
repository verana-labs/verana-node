package ante

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Fence interface {
	CheckMessages(ctx context.Context, msgs []sdk.Msg) error
}

type Decorator struct {
	fence Fence
}

func NewDecorator(fence Fence) Decorator {
	return Decorator{fence: fence}
}

func (d Decorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if err := d.fence.CheckMessages(ctx, tx.GetMsgs()); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}
