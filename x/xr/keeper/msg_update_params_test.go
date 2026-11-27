package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/xr/keeper"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func TestMsgUpdateParams(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	testCases := []struct {
		name   string
		input  *types.MsgUpdateParams
		expErr error
	}{
		{
			name:   "invalid authority",
			input:  &types.MsgUpdateParams{Authority: "invalid", Params: params},
			expErr: types.ErrInvalidSigner,
		},
		{
			name:   "zero twap_window is rejected",
			input:  &types.MsgUpdateParams{Authority: authorityStr, Params: types.Params{}},
			expErr: types.ErrInvalidParams,
		},
		{
			name:   "negative twap_window is rejected",
			input:  &types.MsgUpdateParams{Authority: authorityStr, Params: types.NewParams(-time.Minute)},
			expErr: types.ErrInvalidParams,
		},
		{
			name:  "all good",
			input: &types.MsgUpdateParams{Authority: authorityStr, Params: types.NewParams(15 * time.Minute)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(f.ctx, tc.input)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				return
			}
			require.NoError(t, err)
			got, err := f.keeper.Params.Get(f.ctx)
			require.NoError(t, err)
			require.Equal(t, tc.input.Params, got)
		})
	}
}

func TestMsgUpdateParams_EmitsParamValues(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: authorityStr, Params: types.NewParams(15 * time.Minute)})
	require.NoError(t, err)

	attrs := map[string]string{}
	for _, ev := range sdk.UnwrapSDKContext(f.ctx).EventManager().Events() {
		if ev.Type == types.EventTypeUpdateParams {
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
		}
	}
	require.Equal(t, map[string]string{types.AttributeKeyAuthority: authorityStr, "twap_window": "15m0s"}, attrs)
}
