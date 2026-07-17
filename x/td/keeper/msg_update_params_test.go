package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/td/types"
)

// TestUpdateParams_PreservesShareValue pins TD-CRIT-1: a governance params
// update MUST NOT reset the live trust_deposit_share_value.
func TestUpdateParams_PreservesShareValue(t *testing.T) {
	k, ms, ctx, _ := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Simulate the BeginBlocker having grown the live share value.
	live := types.DefaultParams()
	live.TrustDepositShareValue = math.LegacyMustNewDecFromStr("1.5")
	require.NoError(t, k.SetParams(wctx, live))

	// Governance updates params carrying a stale share value (1.0).
	_, err := ms.UpdateParams(wctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: types.DefaultParams()})
	require.NoError(t, err)

	require.Equal(t, math.LegacyMustNewDecFromStr("1.5"), k.GetParams(wctx).TrustDepositShareValue)
}

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx, _ := setupMsgServer(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	wctx := sdk.UnwrapSDKContext(ctx)

	// default params
	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "send enabled param",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			expErr: false,
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
