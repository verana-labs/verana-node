package keeper_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func TestUpdateExchangeRate_HappyPath(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 0)

	f.at(testNow.Add(time.Minute))
	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint64(200), xr.Rate)
	require.Equal(t, testNow.Unix()+60, xr.Updated)
	require.Equal(t, testNow.Unix()+60+600, xr.Expires)
	require.True(t, xr.State)

	attrs := eventAttrs(t, f, types.EventTypeUpdateExchangeRate)
	require.Equal(t, map[string]string{
		"module":                  types.ModuleName,
		types.AttributeKeyXrID:    "1",
		types.AttributeKeyUpdated: "2023-11-14T22:14:20Z",
		types.AttributeKeyExpires: "2023-11-14T22:24:20Z",
	}, attrs)
}

func TestUpdateExchangeRate_NotFound(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewMsgServerImpl(f.keeper).UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: testAccount("feeder"), Id: 999, Rate: 200})
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)
}

func TestUpdateExchangeRate_NotAuthorized(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)
	grantAuthz(t, f, id, testAccount("feeder"), 0, 0)

	_, err := keeper.NewMsgServerImpl(f.keeper).UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: testAccount("other"), Id: id, Rate: 200})
	require.ErrorIs(t, err, types.ErrAuthorizationNotFound)
}

func TestUpdateExchangeRate_NotActive(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), false)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 0)

	_, err := keeper.NewMsgServerImpl(f.keeper).UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.ErrorIs(t, err, types.ErrExchangeRateNotActive)
}

func TestUpdateExchangeRate_InvalidRate(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 0)

	_, err := keeper.NewMsgServerImpl(f.keeper).UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 0})
	require.ErrorIs(t, err, types.ErrInvalidRate)
}

func TestUpdateExchangeRate_ExpiredStillUpdatable(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 0)

	f.at(testNow.Add(30 * time.Minute))
	_, err := keeper.NewMsgServerImpl(f.keeper).UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, testNow.Add(40*time.Minute).Unix(), xr.Expires)
}

func TestUpdateExchangeRate_AuthzExpired(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 0)

	f.at(testNow.Add(time.Hour))
	_, err := keeper.NewMsgServerImpl(f.keeper).UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.ErrorIs(t, err, types.ErrAuthorizationExpired)
}

func TestUpdateExchangeRate_MinIntervalAntiSpam(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 5*time.Minute, 0)

	f.at(testNow.Add(5*time.Minute - time.Second))
	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.ErrorIs(t, err, types.ErrUpdateTooSoon)

	f.at(testNow.Add(5 * time.Minute))
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.NoError(t, err)
}

func TestUpdateExchangeRate_MinIntervalAnyFeeder(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	first, second := testAccount("feeder"), testAccount("second_feeder")
	grantAuthz(t, f, id, first, 0, 0)
	grantAuthz(t, f, id, second, 5*time.Minute, 0)

	f.at(testNow.Add(10 * time.Minute))
	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: first, Id: id, Rate: 200})
	require.NoError(t, err)

	f.at(testNow.Add(14 * time.Minute))
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: second, Id: id, Rate: 210})
	require.ErrorIs(t, err, types.ErrUpdateTooSoon)
}

func TestUpdateExchangeRate_StateChangeKeepsUpdated(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 5*time.Minute, 0)

	f.at(testNow.Add(4 * time.Minute))
	_, err := ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{Authority: f.authority(t), Id: id, State: false})
	require.NoError(t, err)
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{Authority: f.authority(t), Id: id, State: true})
	require.NoError(t, err)

	f.at(testNow.Add(5 * time.Minute))
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 200})
	require.NoError(t, err)
}

func TestUpdateExchangeRate_MaxDeviationCircuitBreaker(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 1000)

	for _, rate := range []uint64{111, 89} {
		_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: rate})
		require.ErrorIs(t, err, types.ErrRateDeviationExceeded)
	}
	for _, rate := range []uint64{110, 99} {
		_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: rate})
		require.NoError(t, err)
	}
}

func TestUpdateExchangeRate_MaxDeviationNoOverflow(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	msg := createMsg(t, f)
	msg.Rate = math.MaxUint64 / 2
	id := createRate(t, f, msg, true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 10000)

	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: math.MaxUint64 - 1})
	require.NoError(t, err)
}
