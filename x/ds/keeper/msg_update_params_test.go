package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/keeper"
	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestMsgUpdateParams(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authority := f.authority(t)

	withEdit := func(edit func(p *types.Params)) types.Params {
		p := types.DefaultParams()
		edit(&p)
		return p
	}
	testCases := []struct {
		name   string
		input  *types.MsgUpdateParams
		expErr error
	}{
		{
			name:   "invalid authority",
			input:  &types.MsgUpdateParams{Authority: "invalid", Params: types.DefaultParams()},
			expErr: types.ErrInvalidSigner,
		},
		{
			name:   "empty params",
			input:  &types.MsgUpdateParams{Authority: authority, Params: types.Params{}},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "resulting min uptime above target",
			input: &types.MsgUpdateParams{Authority: authority, Params: withEdit(func(p *types.Params) {
				p.ValidatorMinUptime = math.LegacyMustNewDecFromStr("0.995")
			})},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{Authority: authority, Params: withEdit(func(p *types.Params) {
				p.ValidatorNodeBudget = math.LegacyMustNewDecFromStr("5000")
				p.CouncilBudget = math.LegacyMustNewDecFromStr("100000")
				p.FoundationBudget = math.LegacyMustNewDecFromStr("200000")
				p.FoundationAccount = authority
				p.UnbondingPeriod = 14 * 24 * time.Hour
			})},
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

	p := types.DefaultParams()
	p.BondingEnabled = true
	_, err := ms.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: authority, Params: p})
	require.NoError(t, err)

	attrs := map[string]string{}
	for _, ev := range sdk.UnwrapSDKContext(f.ctx).EventManager().Events() {
		if ev.Type == types.EventTypeUpdateParams {
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
		}
	}
	require.Equal(t, authority, attrs[types.AttributeKeyAuthority])
	require.Equal(t, "true", attrs["bonding_enabled"])
	require.Equal(t, p.FoundationAccount, attrs["foundation_account"])
	require.Equal(t, "0.990000000000000000", attrs["validator_target_uptime"])
	require.Len(t, attrs, 10)
}
