package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/cs/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, _, ctx := setupMsgServer(t)
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
			name: "zero values",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			expErr:    true,
			expErrMsg: "must be positive",
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

func TestMsgUpdateParams_EmitsParamValues(t *testing.T) {
	k, ms, _, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	params := types.DefaultParams()
	params.CredentialSchemaPresentationValidityPeriodMaxDays = 30

	_, err := ms.UpdateParams(wctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: params})
	require.NoError(t, err)
	require.Equal(t, params, k.GetParams(wctx))

	attrs := map[string]string{}
	for _, ev := range wctx.EventManager().Events() {
		if ev.Type == types.EventTypeUpdateParams {
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
		}
	}
	require.Equal(t, k.GetAuthority(), attrs[types.AttributeKeyAuthority])
	require.Equal(t, "30", attrs["credential_schema_presentation_validity_period_max_days"])
	require.Equal(t, "8192", attrs["credential_schema_schema_max_size"])
	require.Len(t, attrs, 8)
}
