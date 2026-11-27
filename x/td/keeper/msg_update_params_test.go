package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/td/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx, _ := setupMsgServer(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	wctx := sdk.UnwrapSDKContext(ctx)

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
			input:  &types.MsgUpdateParams{Signer: "invalid", Params: params},
			expErr: types.ErrInvalidSigner,
		},
		{
			name:   "empty params",
			input:  &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: types.Params{}},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "trust_deposit_rate above 1",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.TrustDepositRate = math.LegacyMustNewDecFromStr("1.01")
			})},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "tu_decay_rate above 0.01",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.TuDecayRate = math.LegacyMustNewDecFromStr("0.02")
			})},
			expErr: types.ErrInvalidParams,
		},
		{
			name: "main_fiat_currency is immutable",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.MainFiatCurrency = "EUR"
			})},
			expErr: types.ErrImmutableParam,
		},
		{
			name: "tu_peg_value_genesis is immutable",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.TuPegValueGenesis = math.LegacyNewDec(1000)
			})},
			expErr: types.ErrImmutableParam,
		},
		{
			name: "epoch_length is immutable",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.EpochLength = time.Hour
			})},
			expErr: types.ErrImmutableParam,
		},
		{
			name: "vna_total_supply is immutable",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.VnaTotalSupply = 1
			})},
			expErr: types.ErrImmutableParam,
		},
		{
			name: "mutable values",
			input: &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: withEdit(func(p *types.Params) {
				p.TrustDepositRate = math.LegacyMustNewDecFromStr("0.1")
				p.VuaRewardRate = math.LegacyMustNewDecFromStr("0.02")
				p.NetworkGasPrice = math.LegacyMustNewDecFromStr("0.5")
				p.MaxGasPerMsgType = []types.MaxGasEntry{{MsgTypeUrl: "/verana.td.v1.MsgRepaySlashedTrustDeposit", MaxGas: 200000}}
			})},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			before := k.GetParams(wctx)
			_, err := ms.UpdateParams(wctx, tc.input)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				require.Equal(t, before, k.GetParams(wctx))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.input.Params, k.GetParams(wctx))
		})
	}
}

func TestMsgUpdateParams_EmitsParamValues(t *testing.T) {
	k, ms, ctx, _ := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	require.NoError(t, k.SetParams(wctx, types.DefaultParams()))

	next := types.DefaultParams()
	next.NetworkGasPrice = math.LegacyMustNewDecFromStr("0.5")
	next.TuDecayRate = math.LegacyMustNewDecFromStr("0.002")
	next.MaxGasPerMsgType = []types.MaxGasEntry{{MsgTypeUrl: "/cosmos.group.v1.MsgVote", MaxGas: 100000}}
	_, err := ms.UpdateParams(wctx, &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: next})
	require.NoError(t, err)

	ev := lastEvent(t, wctx, types.EventTypeUpdateParams)
	attrs := map[string]string{}
	for _, a := range ev.Attributes {
		attrs[a.Key] = a.Value
	}
	require.Equal(t, k.GetAuthority(), attrs[types.AttributeKeySigner])
	require.Equal(t, "0.500000000000000000", attrs["network_gas_price"])
	require.Equal(t, "0.001000000000000000", attrs["tu_decay_rate"])
	require.Equal(t, "0.002000000000000000", attrs[types.AttributeKeyPendingTuDecayRate])
	require.Equal(t, `{"/cosmos.group.v1.MsgVote":100000}`, attrs["max_gas_per_msg_type"])
	for _, key := range []string{
		"trust_deposit_rate", "main_fiat_currency", "tu_peg_value_genesis", "epoch_length",
		"vua_reward_rate", "vna_total_supply", "max_gas_per_msg_type",
	} {
		require.Contains(t, attrs, key)
	}
}

func TestMsgUpdateParams_PendingTuDecayRate(t *testing.T) {
	k, ms, ctx, _ := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	require.NoError(t, k.SetParams(wctx, types.DefaultParams()))
	inForce := types.DefaultParams().TuDecayRate

	send := func(p types.Params) {
		_, err := ms.UpdateParams(wctx, &types.MsgUpdateParams{Signer: k.GetAuthority(), Params: p})
		require.NoError(t, err)
	}
	withRate := func(rate string) types.Params {
		p := types.DefaultParams()
		p.TuDecayRate = math.LegacyMustNewDecFromStr(rate)
		return p
	}
	requirePending := func(rate string) {
		pending, found, err := k.GetPendingTuDecayRate(wctx)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, math.LegacyMustNewDecFromStr(rate), pending)
		require.Equal(t, inForce, k.GetParams(wctx).TuDecayRate)
	}

	send(withRate("0.002"))
	requirePending("0.002")

	send(withRate("0.003"))
	requirePending("0.003")

	// queried params carry the rate in force
	p := k.GetParams(wctx)
	p.NetworkGasPrice = math.LegacyMustNewDecFromStr("0.5")
	send(p)
	requirePending("0.003")
	require.Equal(t, p.NetworkGasPrice, k.GetParams(wctx).NetworkGasPrice)
	require.Equal(t, "0.003000000000000000", attrValue(lastEvent(t, wctx, types.EventTypeUpdateParams), types.AttributeKeyPendingTuDecayRate))

	send(withRate(inForce.String()))
	requirePending("0.003")
}

func lastEvent(t *testing.T, ctx sdk.Context, eventType string) sdk.Event {
	t.Helper()
	events := ctx.EventManager().Events()
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == eventType {
			return events[i]
		}
	}
	t.Fatalf("event %s not emitted", eventType)
	return sdk.Event{}
}

func attrValue(ev sdk.Event, key string) string {
	for _, a := range ev.Attributes {
		if a.Key == key {
			return a.Value
		}
	}
	return ""
}
