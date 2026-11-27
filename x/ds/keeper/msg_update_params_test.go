package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/keeper"
	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestMsgUpdateParams(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authority := f.authority(t)
	blocked := sdk.AccAddress(authtypes.NewModuleAddress("bonded_tokens_pool")).String()
	f.bank.blocked[blocked] = true

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
			name:   "invalid signer",
			input:  &types.MsgUpdateParams{Signer: "invalid", Params: types.DefaultParams()},
			expErr: types.ErrInvalidSigner,
		},
		{
			name:   "empty params",
			input:  &types.MsgUpdateParams{Signer: authority, Params: types.Params{}},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "resulting min uptime above target",
			input: &types.MsgUpdateParams{Signer: authority, Params: withEdit(func(p *types.Params) {
				p.ValidatorMinUptime = math.LegacyMustNewDecFromStr("0.995")
			})},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "foundation account blocked",
			input: &types.MsgUpdateParams{Signer: authority, Params: withEdit(func(p *types.Params) {
				p.FoundationAccount = blocked
			})},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{Signer: authority, Params: withEdit(func(p *types.Params) {
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

func TestMsgUpdateParams_EmitsMsg(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	p := types.DefaultParams()
	p.BondingEnabled = true
	msg := &types.MsgUpdateParams{Signer: f.authority(t), Params: p}
	_, err := ms.UpdateParams(f.ctx, msg)
	require.NoError(t, err)

	want, err := f.cdc.MarshalJSON(msg)
	require.NoError(t, err)
	var found bool
	for _, ev := range f.ctx.EventManager().Events() {
		if ev.Type != types.EventTypeUpdateModuleParameters {
			continue
		}
		found = true
		attrs := map[string]string{}
		for _, a := range ev.Attributes {
			attrs[a.Key] = a.Value
		}
		require.Equal(t, map[string]string{sdk.AttributeKeyModule: types.ModuleName, "msg": string(want)}, attrs)
	}
	require.True(t, found)
}
