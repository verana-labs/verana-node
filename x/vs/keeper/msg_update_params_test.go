package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/vs/keeper"
	"github.com/verana-labs/verana-node/x/vs/types"
)

func TestMsgUpdateParams(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authority := f.authority(t)

	valid := types.NewParams(500, 10000, 30*24*time.Hour, 12*time.Hour)
	testCases := []struct {
		name   string
		input  *types.MsgUpdateParams
		expErr error
	}{
		{
			name:   "invalid signer",
			input:  &types.MsgUpdateParams{Signer: "invalid", Params: valid},
			expErr: types.ErrInvalidSigner,
		},
		{
			name:   "empty params",
			input:  &types.MsgUpdateParams{Signer: authority, Params: types.Params{}},
			expErr: types.ErrInvalidParams,
		},
		{
			name:   "vs_account_fee_period zero",
			input:  &types.MsgUpdateParams{Signer: authority, Params: types.NewParams(0, 0, time.Hour, 0)},
			expErr: types.ErrInvalidParams,
		},
		{
			name:  "all good",
			input: &types.MsgUpdateParams{Signer: authority, Params: valid},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			before, err := f.keeper.Params.Get(f.ctx)
			require.NoError(t, err)
			_, err = ms.UpdateParams(f.ctx, tc.input)
			after, gerr := f.keeper.Params.Get(f.ctx)
			require.NoError(t, gerr)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				require.Equal(t, before, after)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.input.Params, after)
		})
	}
}

func TestMsgUpdateParams_EmitsParamValues(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authority := f.authority(t)

	_, err := ms.UpdateParams(f.ctx, &types.MsgUpdateParams{Signer: authority, Params: types.NewParams(500, 10000, time.Hour, 2*time.Hour)})
	require.NoError(t, err)

	attrs := map[string]string{}
	for _, ev := range sdk.UnwrapSDKContext(f.ctx).EventManager().Events() {
		if ev.Type == types.EventTypeUpdateParams {
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
		}
	}
	require.Equal(t, map[string]string{
		types.AttributeKeySigner:      authority,
		"vs_registration_deposit":     "500",
		"ecosystem_creation_deposit":  "10000",
		"vid_rotation_max_transition": "1h0m0s",
		"vs_account_fee_period":       "2h0m0s",
	}, attrs)
}

func TestVsAccountFeePeriod(t *testing.T) {
	f := initFixture(t)
	period, err := f.keeper.VsAccountFeePeriod(f.ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultVsAccountFeePeriod, period)

	require.NoError(t, f.keeper.Params.Set(f.ctx, types.NewParams(0, 0, time.Hour, 6*time.Hour)))
	period, err = f.keeper.VsAccountFeePeriod(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 6*time.Hour, period)
}
