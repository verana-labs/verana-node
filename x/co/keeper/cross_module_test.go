package keeper_test

import (
	"testing"

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

	// SetActiveVersion happy path.
	require.NoError(t, adapter.SetActiveVersion(ctx, 1, 5))
	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(5), co.ActiveVersion)

	// SetActiveVersion on missing corp returns error.
	require.Error(t, adapter.SetActiveVersion(ctx, 999, 5))
}

func TestCoAsDeCorporationKeeper_Sponsor(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1corp"}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	_, err := keeper.NewMsgServerImpl(k).CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)
	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	co.SponsorCorporationId = 7
	require.NoError(t, k.Corporation.Set(ctx, 1, co))

	adapter := keeper.NewCoAsDeCorporationKeeper(k)
	view, found, err := adapter.GetCorporationByPolicyAddress(ctx, "cosmos1corp")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(7), view.SponsorCorporationId)
	_, found, err = adapter.GetCorporationByPolicyAddress(ctx, "cosmos1nobody")
	require.NoError(t, err)
	require.False(t, found)

	cleared, err := adapter.ClearSponsor(ctx, 1, 8)
	require.NoError(t, err)
	require.False(t, cleared)
	cleared, err = adapter.ClearSponsor(ctx, 1, 7)
	require.NoError(t, err)
	require.True(t, cleared)
	view, err = adapter.ResolveCorporationByID(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, view.SponsorCorporationId)
}
