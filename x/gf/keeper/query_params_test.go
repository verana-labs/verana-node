package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/gf/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

func TestQueryParams(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, types.DefaultParams(), resp.Params)
}

// [MOD-GF-QRY-3-1] the network governance group and the version in force.
func TestQueryParamsNetworkGovernance(t *testing.T) {
	grace := types.DefaultGfAcceptanceGraceMin
	k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, k.GetAuthority(), resp.CouncilAccount)
	require.Equal(t, []uint64{1}, resp.NetworkGovernanceFrameworkVersions)
	require.Nil(t, resp.NgfAcceptanceDeadline)
	require.Equal(t, uint32(1), resp.NgfVersion)

	addNetworkVersion(t, k, ctx, 2, 0)
	resp, err = qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2}, resp.NetworkGovernanceFrameworkVersions)
	require.Equal(t, uint32(1), resp.NgfVersion, "a draft is not the version in force")

	_, err = keeper.NewMsgServerImpl(k).IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
		Signer: k.GetAuthority(), Network: true, AcceptanceGrace: grace,
	})
	require.NoError(t, err)
	resp, err = qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, uint32(2), resp.NgfVersion)
	require.Equal(t, genesisTime.Add(grace), *resp.NgfAcceptanceDeadline)
}
