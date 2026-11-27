package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/co/keeper"
	"github.com/verana-labs/verana-node/x/co/types"
)

func TestCoAsGFCorporationKeeper(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1corp"}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	adapter := keeper.NewCoAsGFCorporationKeeper(k)

	// ResolveCorporationByPolicyAddress hit.
	view, err := adapter.ResolveCorporationByPolicyAddress(ctx, "cosmos1corp")
	require.NoError(t, err)
	require.Equal(t, uint64(1), view.Id)
	require.Equal(t, uint32(1), view.ActiveVersion)

	// Miss.
	_, err = adapter.ResolveCorporationByPolicyAddress(ctx, "cosmos1nobody")
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)

	// GetByID hit + miss.
	view, ok := adapter.GetByID(ctx, 1)
	require.True(t, ok)
	require.Equal(t, "cosmos1corp", view.PolicyAddress)
	_, ok = adapter.GetByID(ctx, 999)
	require.False(t, ok)

	// SetActiveVersion happy path: the activation opens the grace period.
	deadline := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, adapter.SetActiveVersion(ctx, 1, 5, deadline))
	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(5), co.ActiveVersion)
	require.Equal(t, deadline, *co.AcceptanceDeadline)

	// A later activation replaces the deadline, whether later or earlier.
	for _, d := range []time.Time{deadline.Add(time.Hour), deadline.Add(-time.Hour)} {
		require.NoError(t, adapter.SetActiveVersion(ctx, 1, 6, d))
		co, err = k.Corporation.Get(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, d, *co.AcceptanceDeadline)
		deadline = d
	}

	// SetActiveVersion on missing corp returns error.
	require.Error(t, adapter.SetActiveVersion(ctx, 999, 5, deadline))

	// SetAcceptedNgfVersion records the network framework acceptance.
	require.NoError(t, adapter.SetAcceptedNgfVersion(ctx, 1, 2))
	view, ok = adapter.GetByID(ctx, 1)
	require.True(t, ok)
	require.Equal(t, uint32(2), view.AcceptedNgfVersion)
	require.Equal(t, deadline, *view.AcceptanceDeadline)
	require.Error(t, adapter.SetAcceptedNgfVersion(ctx, 999, 2))
}

// GetCorporationByPolicyAddress is AUTHZ-CHECK-5 without GF-CHECK-1.
func TestCoAsGFCorporationKeeper_GetCorporationByPolicyAddressSkipsGFCheck1(t *testing.T) {
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{policy: "cosmos1corp"}, gf)
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)
	adapter := keeper.NewCoAsGFCorporationKeeper(k)

	gf.ngfErr = errAuthDenied
	_, err = adapter.ResolveCorporationByPolicyAddress(ctx, "cosmos1corp")
	require.ErrorIs(t, err, errAuthDenied)
	view, err := adapter.GetCorporationByPolicyAddress(ctx, "cosmos1corp")
	require.NoError(t, err)
	require.Equal(t, uint64(1), view.Id)
	_, err = adapter.GetCorporationByPolicyAddress(ctx, "cosmos1nobody")
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
}
