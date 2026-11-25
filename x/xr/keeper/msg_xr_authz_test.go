package keeper_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func authzKey(t *testing.T, id uint64, feeder string) collections.Pair[uint64, sdk.AccAddress] {
	t.Helper()
	addr, err := sdk.AccAddressFromBech32(feeder)
	require.NoError(t, err)
	return collections.Join(id, addr)
}

func grantMsg(t *testing.T, f *fixture, id uint64) *types.MsgGrantExchangeRateAuthorization {
	exp := testNow.Add(time.Hour)
	return &types.MsgGrantExchangeRateAuthorization{
		Authority:  f.authority(t),
		XrId:       id,
		Feeder:     testAccount("feeder"),
		Expiration: &exp,
	}
}

func TestGrantExchangeRateAuthorization_Success(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	msg := grantMsg(t, f, id)
	minInterval := 5 * time.Minute
	msg.MinInterval = &minInterval
	msg.MaxDeviationBps = 500

	_, err := ms.GrantExchangeRateAuthorization(f.ctx, msg)
	require.NoError(t, err)

	stored, err := f.keeper.ExchangeRateAuthorizations.Get(f.ctx, authzKey(t, id, msg.Feeder))
	require.NoError(t, err)
	require.Equal(t, types.ExchangeRateAuthorizationRecord{Expiration: testNow.Add(time.Hour).Unix(), MinInterval: 300, MaxDeviationBps: 500}, stored)

	attrs := eventAttrs(t, f, types.EventTypeGrantExchangeRateAuthorization)
	require.Equal(t, "1", attrs[types.AttributeKeyXrID])
	require.Equal(t, msg.Feeder, attrs[types.AttributeKeyFeeder])
	require.Contains(t, attrs["msg"], `"max_deviation_bps":500`)

	// a second grant replaces the record and unsets what it omits
	msg.MinInterval = nil
	msg.MaxDeviationBps = 0
	_, err = ms.GrantExchangeRateAuthorization(f.ctx, msg)
	require.NoError(t, err)
	stored, err = f.keeper.ExchangeRateAuthorizations.Get(f.ctx, authzKey(t, id, msg.Feeder))
	require.NoError(t, err)
	require.Equal(t, types.ExchangeRateAuthorizationRecord{Expiration: testNow.Add(time.Hour).Unix()}, stored)
}

func TestGrantExchangeRateAuthorization_NonAuthority(t *testing.T) {
	f := initFixture(t)
	msg := grantMsg(t, f, createRate(t, f, createMsg(t, f), true))
	msg.Authority = testAccount("not_council")
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestGrantExchangeRateAuthorization_ExpirationPast(t *testing.T) {
	f := initFixture(t)
	msg := grantMsg(t, f, createRate(t, f, createMsg(t, f), true))
	*msg.Expiration = testNow
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidExpiration)
}

func TestGrantExchangeRateAuthorization_InvalidXrId(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, grantMsg(t, f, 999))
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)
}

func TestGrantExchangeRateAuthorization_MaxDeviationOutOfRange(t *testing.T) {
	f := initFixture(t)
	msg := grantMsg(t, f, createRate(t, f, createMsg(t, f), true))
	msg.MaxDeviationBps = 10001
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidMaxDeviation)
}

func TestGrantExchangeRateAuthorization_InvalidMinInterval(t *testing.T) {
	f := initFixture(t)
	msg := grantMsg(t, f, createRate(t, f, createMsg(t, f), true))
	for _, d := range []time.Duration{0, -time.Second, time.Second / 2, 3 * time.Second / 2, time.Duration(math.MaxUint32+1) * time.Second} {
		msg.MinInterval = &d
		_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidMinInterval, d.String())
	}
}

func TestGrantExchangeRateAuthorization_InvalidFeeder(t *testing.T) {
	f := initFixture(t)
	msg := grantMsg(t, f, createRate(t, f, createMsg(t, f), true))
	msg.Feeder = "feeder"
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
	require.ErrorContains(t, err, "invalid feeder address")
}

func TestGrantExchangeRateAuthorization_MissingExpiration(t *testing.T) {
	f := initFixture(t)
	msg := grantMsg(t, f, createRate(t, f, createMsg(t, f), true))
	msg.Expiration = nil
	_, err := keeper.NewMsgServerImpl(f.keeper).GrantExchangeRateAuthorization(f.ctx, msg)
	require.ErrorContains(t, err, "expiration must be set")
}

func TestGrantExchangeRateAuthorization_CanonicalFeeder(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	id := createRate(t, f, createMsg(t, f), true)
	msg := grantMsg(t, f, id)
	feeder := msg.Feeder
	msg.Feeder = strings.ToUpper(feeder)

	_, err := ms.GrantExchangeRateAuthorization(f.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, feeder, eventAttrs(t, f, types.EventTypeGrantExchangeRateAuthorization)[types.AttributeKeyFeeder])

	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Feeder: feeder, Id: id, Rate: 101})
	require.NoError(t, err)

	_, err = ms.RevokeExchangeRateAuthorization(f.ctx, &types.MsgRevokeExchangeRateAuthorization{Authority: f.authority(t), XrId: id, Feeder: msg.Feeder})
	require.NoError(t, err)
	require.Equal(t, feeder, eventAttrs(t, f, types.EventTypeRevokeExchangeRateAuthorization)[types.AttributeKeyFeeder])
}

func TestGrantExchangeRateAuthorization_GenesisPairNeedsMaxDeviation(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	msg := grantMsg(t, f, createRate(t, f, genesisPairMsg(t, f), true))

	_, err := ms.GrantExchangeRateAuthorization(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidMaxDeviation)

	msg.MaxDeviationBps = 1000
	_, err = ms.GrantExchangeRateAuthorization(f.ctx, msg)
	require.NoError(t, err)

	// another fiat currency priced in the native denom is not the genesis pair
	other := genesisPairMsg(t, f)
	other.BaseAsset = "EUR"
	_, err = ms.GrantExchangeRateAuthorization(f.ctx, grantMsg(t, f, createRate(t, f, other, true)))
	require.NoError(t, err)
}

func TestRevokeExchangeRateAuthorization_Success(t *testing.T) {
	f := initFixture(t)
	id := createRate(t, f, createMsg(t, f), true)
	feeder := testAccount("feeder")
	grantAuthz(t, f, id, feeder, 0, 0)

	_, err := keeper.NewMsgServerImpl(f.keeper).RevokeExchangeRateAuthorization(f.ctx, &types.MsgRevokeExchangeRateAuthorization{Authority: f.authority(t), XrId: id, Feeder: feeder})
	require.NoError(t, err)

	has, err := f.keeper.ExchangeRateAuthorizations.Has(f.ctx, authzKey(t, id, feeder))
	require.NoError(t, err)
	require.False(t, has)

	attrs := eventAttrs(t, f, types.EventTypeRevokeExchangeRateAuthorization)
	require.Equal(t, "1", attrs[types.AttributeKeyXrID])
	require.Equal(t, feeder, attrs[types.AttributeKeyFeeder])
	require.Contains(t, attrs["msg"], feeder)
}

func TestRevokeExchangeRateAuthorization_Missing(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewMsgServerImpl(f.keeper).RevokeExchangeRateAuthorization(f.ctx, &types.MsgRevokeExchangeRateAuthorization{Authority: f.authority(t), XrId: 1, Feeder: testAccount("feeder")})
	require.ErrorIs(t, err, types.ErrAuthorizationNotFound)
}

func TestRevokeExchangeRateAuthorization_NonAuthority(t *testing.T) {
	f := initFixture(t)
	_, err := keeper.NewMsgServerImpl(f.keeper).RevokeExchangeRateAuthorization(f.ctx, &types.MsgRevokeExchangeRateAuthorization{Authority: testAccount("not_council"), XrId: 1, Feeder: testAccount("feeder")})
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}
