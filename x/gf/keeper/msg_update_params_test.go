package keeper_test

import (
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/gf/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

func TestMsgUpdateParams(t *testing.T) {
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	t.Run("happy path with gov authority", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Signer: govAuthority,
			Params: types.DefaultParams(),
		})
		require.NoError(t, err)
	})

	t.Run("rejects zero gf_acceptance_grace_min", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Signer: govAuthority,
			Params: types.NewParams(0),
		})
		require.ErrorIs(t, err, types.ErrInvalidParams)
	})

	t.Run("stores the value and emits it", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Signer: govAuthority,
			Params: types.NewParams(7 * 24 * time.Hour),
		})
		require.NoError(t, err)
		require.Equal(t, types.NewParams(7*24*time.Hour), k.GetParams(ctx))

		attrs := map[string]string{}
		for _, ev := range ctx.EventManager().Events() {
			if ev.Type == types.EventTypeUpdateParams {
				for _, a := range ev.Attributes {
					attrs[a.Key] = a.Value
				}
			}
		}
		require.Equal(t, map[string]string{
			types.AttributeKeySigner:  govAuthority,
			"gf_acceptance_grace_min": "168h0m0s",
		}, attrs)
	})

	t.Run("rejects non-authority signer", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Signer: testCorp, // not the gov module address
			Params: types.DefaultParams(),
		})
		require.Error(t, err)
	})
}
