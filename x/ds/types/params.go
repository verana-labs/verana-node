package types

import (
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const DefaultUnbondingPeriod = 28 * 24 * time.Hour

var (
	DefaultValidatorTargetUptime = math.LegacyMustNewDecFromStr("0.99")
	DefaultValidatorMinUptime    = math.LegacyMustNewDecFromStr("0.90")
)

// NewParams creates a new Params instance.
func NewParams(
	validatorNodeBudget math.LegacyDec,
	validatorTargetUptime math.LegacyDec,
	validatorMinUptime math.LegacyDec,
	councilBudget math.LegacyDec,
	foundationBudget math.LegacyDec,
	foundationAccount string,
	vnaHolderMaxYieldRate math.LegacyDec,
	unbondingPeriod time.Duration,
	bondingEnabled bool,
) Params {
	return Params{
		ValidatorNodeBudget:   validatorNodeBudget,
		ValidatorTargetUptime: validatorTargetUptime,
		ValidatorMinUptime:    validatorMinUptime,
		CouncilBudget:         councilBudget,
		FoundationBudget:      foundationBudget,
		FoundationAccount:     foundationAccount,
		VnaHolderMaxYieldRate: vnaHolderMaxYieldRate,
		UnbondingPeriod:       unbondingPeriod,
		BondingEnabled:        bondingEnabled,
	}
}

// DefaultParams returns the GLO defaults; budgets, yield cap and foundation
// account are placeholders until the governance framework and genesis set them.
func DefaultParams() Params {
	return NewParams(
		math.LegacyZeroDec(),
		DefaultValidatorTargetUptime,
		DefaultValidatorMinUptime,
		math.LegacyZeroDec(),
		math.LegacyZeroDec(),
		sdk.AccAddress(authtypes.NewModuleAddress(FoundationPlaceholder)).String(),
		math.LegacyZeroDec(),
		DefaultUnbondingPeriod,
		false,
	)
}

// Validate checks [MOD-DS-MSG-1-2-1].
func (p Params) Validate() error {
	for _, f := range []struct {
		name string
		v    math.LegacyDec
	}{
		{"validator_node_budget", p.ValidatorNodeBudget},
		{"council_budget", p.CouncilBudget},
		{"foundation_budget", p.FoundationBudget},
		{"vna_holder_max_yield_rate", p.VnaHolderMaxYieldRate},
	} {
		if f.v.IsNil() || f.v.IsNegative() {
			return errorsmod.Wrapf(ErrInvalidParams, "%s must be >= 0", f.name)
		}
	}
	if p.ValidatorMinUptime.IsNil() || p.ValidatorTargetUptime.IsNil() ||
		p.ValidatorMinUptime.IsNegative() ||
		p.ValidatorMinUptime.GT(p.ValidatorTargetUptime) ||
		p.ValidatorTargetUptime.GT(math.LegacyOneDec()) {
		return errorsmod.Wrap(ErrInvalidParams, "0 <= validator_min_uptime <= validator_target_uptime <= 1 must hold")
	}
	if _, err := sdk.AccAddressFromBech32(p.FoundationAccount); err != nil {
		return errorsmod.Wrapf(ErrInvalidParams, "foundation_account must be a valid account: %s", err)
	}
	if p.UnbondingPeriod <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "unbonding_period must be positive")
	}
	return nil
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	return []sdk.Attribute{
		sdk.NewAttribute("validator_node_budget", p.ValidatorNodeBudget.String()),
		sdk.NewAttribute("validator_target_uptime", p.ValidatorTargetUptime.String()),
		sdk.NewAttribute("validator_min_uptime", p.ValidatorMinUptime.String()),
		sdk.NewAttribute("council_budget", p.CouncilBudget.String()),
		sdk.NewAttribute("foundation_budget", p.FoundationBudget.String()),
		sdk.NewAttribute("foundation_account", p.FoundationAccount),
		sdk.NewAttribute("vna_holder_max_yield_rate", p.VnaHolderMaxYieldRate.String()),
		sdk.NewAttribute("unbonding_period", p.UnbondingPeriod.String()),
		sdk.NewAttribute("bonding_enabled", strconv.FormatBool(p.BondingEnabled)),
	}
}
