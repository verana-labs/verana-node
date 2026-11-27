package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/td/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestMaxGasMsgTypeURLs(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	params := types.DefaultParams()
	params.MaxGasPerMsgType = []types.MaxGasEntry{
		{MsgTypeUrl: "/verana.ec.v1.MsgCreateEcosystem", MaxGas: 300000},
		{MsgTypeUrl: "/cosmos.group.v1.MsgVote", MaxGas: 100000},
	}
	require.NoError(t, k.SetParams(ctx, params))

	urls, err := k.MaxGasMsgTypeURLs(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"/verana.ec.v1.MsgCreateEcosystem", "/cosmos.group.v1.MsgVote"}, urls)
}
