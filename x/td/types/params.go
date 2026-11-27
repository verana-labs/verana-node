package types

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultMainFiatCurrency = "USD"
	DefaultEpochLength      = 24 * time.Hour
	DefaultVnaTotalSupply   = uint64(1_000_000_000_000_000)
)

var (
	DefaultTrustDepositRate  = math.LegacyMustNewDecFromStr("0.05")
	DefaultTuPegValueGenesis = math.LegacyNewDec(100)
	DefaultTuDecayRate       = math.LegacyMustNewDecFromStr("0.001")
	DefaultVuaRewardRate     = math.LegacyMustNewDecFromStr("0.05")
	// placeholder; the governance framework defines the network gas price
	DefaultNetworkGasPrice = math.LegacyMustNewDecFromStr("0.00001")

	MaxTuDecayRate = math.LegacyMustNewDecFromStr("0.01")

	iso4217Code = regexp.MustCompile(`^[A-Z]{3}$`)
)

// NewParams creates a new Params instance.
func NewParams(
	trustDepositRate math.LegacyDec,
	mainFiatCurrency string,
	tuPegValueGenesis math.LegacyDec,
	tuDecayRate math.LegacyDec,
	epochLength time.Duration,
	vuaRewardRate math.LegacyDec,
	vnaTotalSupply uint64,
	networkGasPrice math.LegacyDec,
	maxGasPerMsgType []MaxGasEntry,
) Params {
	return Params{
		TrustDepositRate:  trustDepositRate,
		MainFiatCurrency:  mainFiatCurrency,
		TuPegValueGenesis: tuPegValueGenesis,
		TuDecayRate:       tuDecayRate,
		EpochLength:       epochLength,
		VuaRewardRate:     vuaRewardRate,
		VnaTotalSupply:    vnaTotalSupply,
		NetworkGasPrice:   networkGasPrice,
		MaxGasPerMsgType:  maxGasPerMsgType,
	}
}

// DefaultParams returns the GLO defaults.
func DefaultParams() Params {
	return NewParams(
		DefaultTrustDepositRate,
		DefaultMainFiatCurrency,
		DefaultTuPegValueGenesis,
		DefaultTuDecayRate,
		DefaultEpochLength,
		DefaultVuaRewardRate,
		DefaultVnaTotalSupply,
		DefaultNetworkGasPrice,
		nil,
	)
}

// Validate checks the ranges of [MOD-TD-MSG-4-2-1] and the genesis-only values.
func (p Params) Validate() error {
	if err := validateRate("trust_deposit_rate", p.TrustDepositRate, math.LegacyOneDec()); err != nil {
		return err
	}
	if !iso4217Code.MatchString(p.MainFiatCurrency) {
		return errorsmod.Wrapf(ErrInvalidParams, "main_fiat_currency must be an ISO-4217 code: %q", p.MainFiatCurrency)
	}
	if err := validatePowerOfTen("tu_peg_value_genesis", p.TuPegValueGenesis); err != nil {
		return err
	}
	if err := validateRate("tu_decay_rate", p.TuDecayRate, MaxTuDecayRate); err != nil {
		return err
	}
	if p.EpochLength <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "epoch_length must be positive")
	}
	if err := validateRate("vua_reward_rate", p.VuaRewardRate, math.LegacyOneDec()); err != nil {
		return err
	}
	if p.VnaTotalSupply == 0 {
		return errorsmod.Wrap(ErrInvalidParams, "vna_total_supply must be positive")
	}
	if p.NetworkGasPrice.IsNil() || !p.NetworkGasPrice.IsPositive() {
		return errorsmod.Wrap(ErrInvalidParams, "network_gas_price must be positive")
	}
	seen := make(map[string]bool, len(p.MaxGasPerMsgType))
	for _, e := range p.MaxGasPerMsgType {
		if !strings.HasPrefix(e.MsgTypeUrl, "/") || len(e.MsgTypeUrl) < 2 {
			return errorsmod.Wrapf(ErrInvalidParams, "max_gas_per_msg_type key must be a type URL: %q", e.MsgTypeUrl)
		}
		if seen[e.MsgTypeUrl] {
			return errorsmod.Wrapf(ErrInvalidParams, "max_gas_per_msg_type has %s twice", e.MsgTypeUrl)
		}
		seen[e.MsgTypeUrl] = true
		if e.MaxGas == 0 {
			return errorsmod.Wrapf(ErrInvalidParams, "max_gas_per_msg_type value for %s must be positive", e.MsgTypeUrl)
		}
	}
	return nil
}

// CheckImmutable refuses a change of the immutable parameters.
func (p Params) CheckImmutable(current Params) error {
	switch {
	case p.MainFiatCurrency != current.MainFiatCurrency:
		return errorsmod.Wrap(ErrImmutableParam, "main_fiat_currency")
	case p.TuPegValueGenesis.IsNil() || !p.TuPegValueGenesis.Equal(current.TuPegValueGenesis):
		return errorsmod.Wrap(ErrImmutableParam, "tu_peg_value_genesis")
	case p.EpochLength != current.EpochLength:
		return errorsmod.Wrap(ErrImmutableParam, "epoch_length")
	case p.VnaTotalSupply != current.VnaTotalSupply:
		return errorsmod.Wrap(ErrImmutableParam, "vna_total_supply")
	}
	return nil
}

// ValidateTuDecayRate checks the tu_decay_rate range, also used for the pending rate.
func ValidateTuDecayRate(rate math.LegacyDec) error {
	return validateRate("tu_decay_rate", rate, MaxTuDecayRate)
}

func validateRate(name string, v, max math.LegacyDec) error {
	if v.IsNil() || v.IsNegative() || v.GT(max) {
		return errorsmod.Wrapf(ErrInvalidParams, "%s must be in [0, %s]: %s", name, max, decString(v))
	}
	return nil
}

func validatePowerOfTen(name string, v math.LegacyDec) error {
	if v.IsNil() || !v.IsPositive() || !v.IsInteger() {
		return errorsmod.Wrapf(ErrInvalidParams, "%s must be a power of ten: %s", name, decString(v))
	}
	s := v.TruncateInt().String()
	if s[0] != '1' || strings.Trim(s[1:], "0") != "" {
		return errorsmod.Wrapf(ErrInvalidParams, "%s must be a power of ten: %s", name, s)
	}
	return nil
}

func decString(v math.LegacyDec) string {
	if v.IsNil() {
		return "<nil>"
	}
	return v.String()
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	m := make(map[string]uint64, len(p.MaxGasPerMsgType))
	for _, e := range p.MaxGasPerMsgType {
		m[e.MsgTypeUrl] = e.MaxGas
	}
	maxGas, _ := json.Marshal(m)
	return []sdk.Attribute{
		sdk.NewAttribute("trust_deposit_rate", p.TrustDepositRate.String()),
		sdk.NewAttribute("main_fiat_currency", p.MainFiatCurrency),
		sdk.NewAttribute("tu_peg_value_genesis", p.TuPegValueGenesis.String()),
		sdk.NewAttribute("tu_decay_rate", p.TuDecayRate.String()),
		sdk.NewAttribute("epoch_length", p.EpochLength.String()),
		sdk.NewAttribute("vua_reward_rate", p.VuaRewardRate.String()),
		sdk.NewAttribute("vna_total_supply", strconv.FormatUint(p.VnaTotalSupply, 10)),
		sdk.NewAttribute("network_gas_price", p.NetworkGasPrice.String()),
		sdk.NewAttribute("max_gas_per_msg_type", string(maxGas)),
	}
}
