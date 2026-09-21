package ante

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Fence interface {
	CheckMessages(ctx context.Context, msgs []sdk.Msg, inGenesis bool) error
}

type Decorator struct {
	inGenesis func() bool
	fence     Fence
}

func NewDecorator(inGenesis func() bool, fence Fence) Decorator {
	return Decorator{inGenesis: inGenesis, fence: fence}
}

func (d Decorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if err := d.fence.CheckMessages(ctx, tx.GetMsgs(), d.inGenesis()); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}
