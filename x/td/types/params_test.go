package types_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/td/types"
)

func TestParams_ValidateRejectsNilDec(t *testing.T) {
	require.Error(t, types.Params{}.Validate())

	p := types.DefaultParams()
	p.TrustDepositRate = math.LegacyDec{}
	require.Error(t, p.Validate())
}

func TestParams_DefaultsMatchGLO(t *testing.T) {
	p := types.DefaultParams()
	require.Equal(t, math.LegacyMustNewDecFromStr("0.05"), p.TrustDepositRate)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.001"), p.TuDecayRate)
	require.Equal(t, 24*time.Hour, p.EpochLength)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.05"), p.VuaRewardRate)
	require.Equal(t, uint64(1_000_000_000_000_000), p.VnaTotalSupply)
	require.Equal(t, math.LegacyNewDec(100), p.TuPegValueGenesis)
	require.NoError(t, p.Validate())
}

func TestParams_Validate(t *testing.T) {
	dec := math.LegacyMustNewDecFromStr
	tests := []struct {
		name  string
		edit  func(p *types.Params)
		valid bool
	}{
		{"default", func(p *types.Params) {}, true},
		{"trust_deposit_rate 0", func(p *types.Params) { p.TrustDepositRate = dec("0") }, true},
		{"trust_deposit_rate 1", func(p *types.Params) { p.TrustDepositRate = dec("1") }, true},
		{"trust_deposit_rate above 1", func(p *types.Params) { p.TrustDepositRate = dec("1.000000000000000001") }, false},
		{"trust_deposit_rate negative", func(p *types.Params) { p.TrustDepositRate = dec("-0.1") }, false},
		{"vua_reward_rate 1", func(p *types.Params) { p.VuaRewardRate = dec("1") }, true},
		{"vua_reward_rate above 1", func(p *types.Params) { p.VuaRewardRate = dec("1.1") }, false},
		{"vua_reward_rate negative", func(p *types.Params) { p.VuaRewardRate = dec("-0.01") }, false},
		{"tu_decay_rate 0", func(p *types.Params) { p.TuDecayRate = dec("0") }, true},
		{"tu_decay_rate 0.01", func(p *types.Params) { p.TuDecayRate = dec("0.01") }, true},
		{"tu_decay_rate above 0.01", func(p *types.Params) { p.TuDecayRate = dec("0.010000000000000001") }, false},
		{"tu_decay_rate negative", func(p *types.Params) { p.TuDecayRate = dec("-0.001") }, false},
		{"network_gas_price zero", func(p *types.Params) { p.NetworkGasPrice = dec("0") }, false},
		{"network_gas_price negative", func(p *types.Params) { p.NetworkGasPrice = dec("-1") }, false},
		{"max_gas value zero", func(p *types.Params) {
			p.MaxGasPerMsgType = []types.MaxGasEntry{{MsgTypeUrl: "/verana.cs.v1.MsgCreateCredentialSchema", MaxGas: 0}}
		}, false},
		{"max_gas positive", func(p *types.Params) {
			p.MaxGasPerMsgType = []types.MaxGasEntry{
				{MsgTypeUrl: "/verana.cs.v1.MsgCreateCredentialSchema", MaxGas: 300000},
				{MsgTypeUrl: "/cosmos.group.v1.MsgVote", MaxGas: 100000},
			}
		}, true},
		{"max_gas key not a type URL", func(p *types.Params) {
			p.MaxGasPerMsgType = []types.MaxGasEntry{{MsgTypeUrl: "MsgCreateCredentialSchema", MaxGas: 300000}}
		}, false},
		{"max_gas duplicate key", func(p *types.Params) {
			p.MaxGasPerMsgType = []types.MaxGasEntry{
				{MsgTypeUrl: "/cosmos.group.v1.MsgVote", MaxGas: 100000},
				{MsgTypeUrl: "/cosmos.group.v1.MsgVote", MaxGas: 200000},
			}
		}, false},
		{"main_fiat_currency lower case", func(p *types.Params) { p.MainFiatCurrency = "usd" }, false},
		{"main_fiat_currency empty", func(p *types.Params) { p.MainFiatCurrency = "" }, false},
		{"main_fiat_currency four letters", func(p *types.Params) { p.MainFiatCurrency = "USDT" }, false},
		{"main_fiat_currency EUR", func(p *types.Params) { p.MainFiatCurrency = "EUR" }, true},
		{"tu_peg_value_genesis 1", func(p *types.Params) { p.TuPegValueGenesis = dec("1") }, true},
		{"tu_peg_value_genesis 1000", func(p *types.Params) { p.TuPegValueGenesis = dec("1000") }, true},
		{"tu_peg_value_genesis zero", func(p *types.Params) { p.TuPegValueGenesis = dec("0") }, false},
		{"tu_peg_value_genesis not a power of ten", func(p *types.Params) { p.TuPegValueGenesis = dec("50") }, false},
		{"tu_peg_value_genesis fraction", func(p *types.Params) { p.TuPegValueGenesis = dec("0.1") }, false},
		{"epoch_length zero", func(p *types.Params) { p.EpochLength = 0 }, false},
		{"vna_total_supply zero", func(p *types.Params) { p.VnaTotalSupply = 0 }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := types.DefaultParams()
			tc.edit(&p)
			if tc.valid {
				require.NoError(t, p.Validate())
			} else {
				require.Error(t, p.Validate())
			}
		})
	}
}

func TestParams_CheckImmutable(t *testing.T) {
	cur := types.DefaultParams()
	tests := []struct {
		name  string
		edit  func(p *types.Params)
		valid bool
	}{
		{"unchanged", func(p *types.Params) {}, true},
		{"mutable change", func(p *types.Params) { p.TrustDepositRate = math.LegacyMustNewDecFromStr("0.1") }, true},
		{"main_fiat_currency", func(p *types.Params) { p.MainFiatCurrency = "EUR" }, false},
		{"tu_peg_value_genesis", func(p *types.Params) { p.TuPegValueGenesis = math.LegacyNewDec(1000) }, false},
		{"epoch_length", func(p *types.Params) { p.EpochLength = time.Hour }, false},
		{"vna_total_supply", func(p *types.Params) { p.VnaTotalSupply = 1 }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := types.DefaultParams()
			tc.edit(&next)
			err := next.CheckImmutable(cur)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, types.ErrImmutableParam)
			}
		})
	}
}
