package ante_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/verana-labs/verana-node/x/poa/ante"
)

type stubTx struct{ msgs []sdk.Msg }

func (t stubTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t stubTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

type stubFence struct {
	seen      []sdk.Msg
	inGenesis bool
	err       error
}

func (s *stubFence) CheckMessages(_ context.Context, msgs []sdk.Msg, inGenesis bool) error {
	s.seen, s.inGenesis = msgs, inGenesis
	return s.err
}

func TestDecorator_ForwardsMessagesAndGenesisFlag(t *testing.T) {
	fence := &stubFence{}
	d := ante.NewDecorator(func() bool { return true }, fence)
	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())
	nextCalled := false
	_, err := d.AnteHandle(ctx, stubTx{[]sdk.Msg{&banktypes.MsgSend{}}}, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { nextCalled = true; return ctx, nil })
	require.NoError(t, err)
	require.True(t, nextCalled)
	require.Len(t, fence.seen, 1)
	require.True(t, fence.inGenesis)
}

func TestDecorator_StopsOnFenceError(t *testing.T) {
	fence := &stubFence{err: errors.New("fenced")}
	d := ante.NewDecorator(func() bool { return false }, fence)
	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())
	_, err := d.AnteHandle(ctx, stubTx{[]sdk.Msg{&banktypes.MsgSend{}}}, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		t.Fatal("next must not run")
		return ctx, nil
	})
	require.ErrorContains(t, err, "fenced")
}
