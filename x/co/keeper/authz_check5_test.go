package keeper_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/co/keeper"
	"github.com/verana-labs/verana-node/x/co/types"
)

func TestAuthzCheck5_UnregisteredAccount(t *testing.T) {
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, gf)

	_, err := k.ResolveCorporationByPolicyAddress(ctx, tkCorp)
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
	require.ErrorContains(t, err, tkCorp)
	require.Zero(t, gf.ngfCalls)
}

func TestAuthzCheck5_RunsGFCheck1AtBlockTime(t *testing.T) {
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{policy: tkPolicy}, gf)
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)

	co, err := k.ResolveCorporationByPolicyAddress(ctx, tkPolicy)
	require.NoError(t, err)
	require.Equal(t, uint64(1), co.Id)
	require.Equal(t, 1, gf.ngfCalls)
	require.Equal(t, uint64(1), gf.ngfCo.Id)
	require.Equal(t, tkPolicy, gf.ngfCo.PolicyAddress)
	require.Equal(t, co.ActiveVersion, gf.ngfCo.ActiveVersion)
	require.Equal(t, now, gf.ngfNow)
}

func TestAuthzCheck5_GFCheck1Aborts(t *testing.T) {
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{policy: tkPolicy}, gf)
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)
	gf.ngfErr = errors.New("accept network governance framework version 2")

	_, err = k.ResolveCorporationByPolicyAddress(ctx, tkPolicy)
	require.ErrorIs(t, err, gf.ngfErr)
}

func TestResolveSigner(t *testing.T) {
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{policy: tkPolicy}, gf)
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	co, group, err := k.ResolveSigner(ctx, tkPolicy, tkPolicy)
	require.NoError(t, err)
	require.True(t, group)
	require.Equal(t, uint64(1), co.Id)

	co, group, err = k.ResolveSigner(ctx, tkPolicy, tkOp)
	require.NoError(t, err)
	require.False(t, group)
	require.Equal(t, uint64(1), co.Id)
	require.Equal(t, 2, gf.ngfCalls)

	_, _, err = k.ResolveSigner(ctx, tkCorp, tkCorp)
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)

	gf.ngfErr = errors.New("accept network governance framework version 2")
	_, _, err = k.ResolveSigner(ctx, tkPolicy, tkPolicy)
	require.ErrorIs(t, err, gf.ngfErr)
}

func TestGetCorporationByPolicyAddress_NoGFCheck1(t *testing.T) {
	gf := &mockGF{ngfErr: errors.New("not accepted")}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{policy: tkPolicy}, gf)
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	co, err := k.GetCorporationByPolicyAddress(ctx, tkPolicy)
	require.NoError(t, err)
	require.Equal(t, uint64(1), co.Id)
	require.Zero(t, gf.ngfCalls)

	_, err = k.GetCorporationByPolicyAddress(ctx, tkCorp)
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
}

// The adapters other modules resolve through run the same check.
func TestAuthzCheck5_AdaptersRunGFCheck1(t *testing.T) {
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{policy: tkPolicy}, gf)
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)
	gf.ngfErr = errors.New("not accepted")

	_, err = keeper.NewCoAsGFCorporationKeeper(k).ResolveCorporationByPolicyAddress(ctx, tkPolicy)
	require.ErrorIs(t, err, gf.ngfErr)
	_, err = keeper.NewCoAsDeCorporationKeeper(k).ResolveCorporationByPolicyAddress(ctx, tkPolicy)
	require.ErrorIs(t, err, gf.ngfErr)
	require.Equal(t, 2, gf.ngfCalls)
}
