package keeper_test

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/keeper"
)

var errNotAccepted = errors.New("not accepted")

type feeTxStub struct {
	sdk.FeeTx
	granter []byte
}

func (s feeTxStub) FeeGranter() []byte { return s.granter }

type plainTxStub struct{ sdk.Tx }

// anteCtx returns the context NewFeeGranterAnteHandler hands to the next handler.
func anteCtx(t *testing.T, ctx sdk.Context, tx sdk.Tx) sdk.Context {
	t.Helper()
	var got sdk.Context
	_, err := keeper.NewFeeGranterAnteHandler(func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		got = c
		return c, nil
	})(ctx, tx, false)
	require.NoError(t, err)
	return got
}

func TestFeeGranterAnteHandler_CallsNextOnce(t *testing.T) {
	ctx := sdk.UnwrapSDKContext(initFixture(t).ctx)
	tx := feeTxStub{granter: sdk.AccAddress([]byte("corp_policy_address_"))}
	nextErr := errors.New("next failed")

	var calls int
	var gotTx sdk.Tx
	var gotSimulate bool
	out, err := keeper.NewFeeGranterAnteHandler(func(c sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		calls++
		gotTx, gotSimulate = tx, simulate
		return c.WithBlockHeight(7), nextErr
	})(ctx, tx, true)

	require.ErrorIs(t, err, nextErr)
	require.Equal(t, 1, calls)
	require.Equal(t, tx, gotTx)
	require.True(t, gotSimulate)
	require.EqualValues(t, 7, out.BlockHeight())
}

func TestFeeGranterFromContext(t *testing.T) {
	ctx := sdk.UnwrapSDKContext(initFixture(t).ctx)
	granter := sdk.AccAddress([]byte("corp_policy_address_"))

	require.Nil(t, keeper.FeeGranterFromContext(ctx))
	require.Equal(t, []byte(granter), keeper.FeeGranterFromContext(anteCtx(t, ctx, feeTxStub{granter: granter})))
	require.Nil(t, keeper.FeeGranterFromContext(anteCtx(t, ctx, plainTxStub{})))
}
