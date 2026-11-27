package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/vs/keeper"
	"github.com/verana-labs/verana-node/x/vs/types"
)

func TestParamsQuery(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.Params(f.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: types.DefaultParams()}, resp)

	_, err = qs.Params(f.ctx, nil)
	require.Error(t, err)
}
