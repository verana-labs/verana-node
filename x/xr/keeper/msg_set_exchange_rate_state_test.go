package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func TestSetExchangeRateState_HappyPath_Enable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), false)

	f.at(testNow.Add(time.Minute))
	_, err := ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{Authority: f.authority(t), Id: id, State: true})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)
	require.Equal(t, testNow.Unix(), xr.Updated)

	attrs := eventAttrs(t, f, types.EventTypeSetExchangeRateState)
	require.Equal(t, "1", attrs[types.AttributeKeyXrID])
	require.Contains(t, attrs["msg"], `"state":true`)
}

func TestSetExchangeRateState_HappyPath_Disable(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)

	_, err := keeper.NewMsgServerImpl(f.keeper).SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{Authority: f.authority(t), Id: id, State: false})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)
}

func TestSetExchangeRateState_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), false)

	_, err := keeper.NewMsgServerImpl(f.keeper).SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{Authority: testAccount("not_council"), Id: id, State: true})
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestSetExchangeRateState_NotFound(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewMsgServerImpl(f.keeper).SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{Authority: f.authority(t), Id: 999, State: true})
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)
}
