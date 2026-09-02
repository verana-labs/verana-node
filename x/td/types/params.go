package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultTrustDepositShareValue    = "1.0"
	DefaultTrustDepositRate          = "0.05" // 5%
	DefaultWalletUserAgentRewardRate = "0.1"  // 10% ([GLO])
	DefaultUserAgentRewardRate       = "0.1"  // 10% ([GLO])
	DefaultTrustDepositMaxYieldRate  = "0.2"  // 20% annual yield ([GLO])
	// [GLO] value; the block-reward share is realized by the protocolpool
	// continuous fund, not read directly here.
	DefaultTrustDepositBlockRewardShare = "0.2"
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	trustDepositShareValue math.LegacyDec,
	trustDepositRate math.LegacyDec,
	walletUserAgentRewardRate math.LegacyDec,
	userAgentRewardRate math.LegacyDec,
	trustDepositMaxYieldRate math.LegacyDec,
	trustDepositBlockRewardShare math.LegacyDec,
) Params {
	return Params{
		TrustDepositShareValue:       trustDepositShareValue,
		TrustDepositRate:             trustDepositRate,
		WalletUserAgentRewardRate:    walletUserAgentRewardRate,
		UserAgentRewardRate:          userAgentRewardRate,
		TrustDepositMaxYieldRate:     trustDepositMaxYieldRate,
		TrustDepositBlockRewardShare: trustDepositBlockRewardShare,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	TrustDepositShareValue, _ := math.LegacyNewDecFromStr(DefaultTrustDepositShareValue)
	TrustDepositRate, _ := math.LegacyNewDecFromStr(DefaultTrustDepositRate)
	WalletUserAgentRewardRate, _ := math.LegacyNewDecFromStr(DefaultWalletUserAgentRewardRate)
	UserAgentRewardRate, _ := math.LegacyNewDecFromStr(DefaultUserAgentRewardRate)
	TrustDepositMaxYieldRate, _ := math.LegacyNewDecFromStr(DefaultTrustDepositMaxYieldRate)
	TrustDepositBlockRewardShare, _ := math.LegacyNewDecFromStr(DefaultTrustDepositBlockRewardShare)

	return NewParams(
		TrustDepositShareValue,
		TrustDepositRate,
		WalletUserAgentRewardRate,
		UserAgentRewardRate,
		TrustDepositMaxYieldRate,
		TrustDepositBlockRewardShare,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositShareValue"),
			&p.TrustDepositShareValue,
			validatePositiveLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositRate"),
			&p.TrustDepositRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("WalletUserAgentRewardRate"),
			&p.WalletUserAgentRewardRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("UserAgentRewardRate"),
			&p.UserAgentRewardRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositMaxYieldRate"),
			&p.TrustDepositMaxYieldRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositBlockRewardShare"),
			&p.TrustDepositBlockRewardShare,
			validateLegacyDec,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validatePositiveLegacyDec(p.TrustDepositShareValue); err != nil {
		return err
	}
	if err := validateLegacyDec(p.TrustDepositRate); err != nil {
		return err
	}
	if err := validateLegacyDec(p.WalletUserAgentRewardRate); err != nil {
		return err
	}
	if err := validateLegacyDec(p.UserAgentRewardRate); err != nil {
		return err
	}
	if err := validateLegacyDec(p.TrustDepositMaxYieldRate); err != nil {
		return err
	}
	if err := validateLegacyDec(p.TrustDepositBlockRewardShare); err != nil {
		return err
	}
	return nil
}

// validateLegacyDec validates that the parameter is a valid decimal between 0 and 1
func validateLegacyDec(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("value cannot be nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("value cannot be negative: %s", v)
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("value cannot be greater than 1: %s", v)
	}

	return nil
}

// validatePositiveLegacyDec validates that the parameter is a positive decimal
func validatePositiveLegacyDec(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("value cannot be nil")
	}

	if v.IsNegative() || v.IsZero() {
		return fmt.Errorf("value must be positive: %s", v)
	}

	return nil
}
