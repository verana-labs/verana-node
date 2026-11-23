package types_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/types"
)

func TestParams_DefaultsMatchGLO(t *testing.T) {
	p := types.DefaultParams()
	require.Equal(t, math.LegacyMustNewDecFromStr("0.99"), p.ValidatorTargetUptime)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.90"), p.ValidatorMinUptime)
	require.Equal(t, 28*24*time.Hour, p.UnbondingPeriod)
	require.False(t, p.BondingEnabled)
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
		{"budgets set", func(p *types.Params) {
			p.ValidatorNodeBudget, p.CouncilBudget, p.FoundationBudget = dec("5000"), dec("100000"), dec("200000")
		}, true},
		{"validator_node_budget negative", func(p *types.Params) { p.ValidatorNodeBudget = dec("-1") }, false},
		{"council_budget negative", func(p *types.Params) { p.CouncilBudget = dec("-1") }, false},
		{"foundation_budget negative", func(p *types.Params) { p.FoundationBudget = dec("-1") }, false},
		{"foundation_budget nil", func(p *types.Params) { p.FoundationBudget = math.LegacyDec{} }, false},
		{"uptimes equal", func(p *types.Params) { p.ValidatorMinUptime, p.ValidatorTargetUptime = dec("0.95"), dec("0.95") }, true},
		{"uptimes 0 and 1", func(p *types.Params) { p.ValidatorMinUptime, p.ValidatorTargetUptime = dec("0"), dec("1") }, true},
		{"min above target", func(p *types.Params) { p.ValidatorMinUptime, p.ValidatorTargetUptime = dec("0.99"), dec("0.9") }, false},
		{"target above 1", func(p *types.Params) { p.ValidatorTargetUptime = dec("1.01") }, false},
		{"min negative", func(p *types.Params) { p.ValidatorMinUptime = dec("-0.1") }, false},
		{"foundation_account invalid", func(p *types.Params) { p.FoundationAccount = "not-an-account" }, false},
		{"foundation_account empty", func(p *types.Params) { p.FoundationAccount = "" }, false},
		{"vna_holder_max_yield_rate negative", func(p *types.Params) { p.VnaHolderMaxYieldRate = dec("-0.01") }, false},
		{"vna_holder_max_yield_rate above 1", func(p *types.Params) { p.VnaHolderMaxYieldRate = dec("1.5") }, true},
		{"unbonding_period zero", func(p *types.Params) { p.UnbondingPeriod = 0 }, false},
		{"bonding_enabled", func(p *types.Params) { p.BondingEnabled = true }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := types.DefaultParams()
			tc.edit(&p)
			if tc.valid {
				require.NoError(t, p.Validate())
			} else {
				require.ErrorIs(t, p.Validate(), types.ErrInvalidParams)
			}
		})
	}
}
