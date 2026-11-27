package types

import (
	"fmt"
	"math"
	"regexp"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
)

const (
	minValidityDuration = time.Minute
	maxDeviationBps     = 10000
	// maxDuration is the longest duration a record holds, in uint32 seconds.
	maxDuration = time.Duration(math.MaxUint32) * time.Second
)

var iso4217Regex = regexp.MustCompile(`^[A-Z]{3}$`)

// ValidateBasic performs stateless validation on MsgCreateExchangeRate.
func (msg *MsgCreateExchangeRate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if err := validatePair(msg.BaseAssetType, msg.BaseAsset, msg.QuoteAssetType, msg.QuoteAsset); err != nil {
		return err
	}
	return validateRate(msg.Rate, msg.RateScale, msg.ValidityDuration)
}

// ValidateBasic performs stateless validation on MsgUpdateExchangeRate.
func (msg *MsgUpdateExchangeRate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if msg.Id == 0 {
		return fmt.Errorf("id must be greater than 0")
	}
	if msg.Rate == 0 {
		return errorsmod.Wrap(ErrInvalidRate, "rate must be greater than 0")
	}
	return nil
}

// ValidateBasic performs stateless validation on MsgSetExchangeRateState.
func (msg *MsgSetExchangeRateState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if msg.Id == 0 {
		return fmt.Errorf("id must be greater than 0")
	}
	return nil
}

// ValidateBasic performs stateless validation on MsgGrantExchangeRateAuthorization.
func (msg *MsgGrantExchangeRateAuthorization) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Feeder); err != nil {
		return fmt.Errorf("invalid feeder address: %w", err)
	}
	if msg.XrId == 0 {
		return fmt.Errorf("xr_id must be greater than 0")
	}
	if msg.Expiration == nil {
		return fmt.Errorf("expiration must be set")
	}
	var minInterval time.Duration
	if msg.MinInterval != nil {
		if *msg.MinInterval <= 0 {
			return ErrInvalidMinInterval
		}
		minInterval = *msg.MinInterval
	}
	return validateBounds(minInterval, msg.MaxDeviationBps)
}

// ValidateBasic performs stateless validation on MsgRevokeExchangeRateAuthorization.
func (msg *MsgRevokeExchangeRateAuthorization) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Feeder); err != nil {
		return fmt.Errorf("invalid feeder address: %w", err)
	}
	if msg.XrId == 0 {
		return fmt.Errorf("xr_id must be greater than 0")
	}
	return nil
}

// ValidateAsset checks a pricing asset: a denom for COIN, an ISO-4217 code for FIAT.
func ValidateAsset(at cstypes.PricingAssetType, asset string) error {
	switch at {
	case cstypes.PricingAssetType_COIN:
		if err := sdk.ValidateDenom(asset); err != nil {
			return errorsmod.Wrapf(ErrInvalidAsset, "%q: %s", asset, err)
		}
	case cstypes.PricingAssetType_FIAT:
		if !iso4217Regex.MatchString(asset) {
			return errorsmod.Wrapf(ErrInvalidAsset, "%q is not an ISO-4217 currency code", asset)
		}
	default:
		return errorsmod.Wrapf(ErrInvalidAssetType, "%s is not COIN or FIAT", at)
	}
	return nil
}

func validatePair(baseType cstypes.PricingAssetType, base string, quoteType cstypes.PricingAssetType, quote string) error {
	if err := ValidateAsset(baseType, base); err != nil {
		return errorsmod.Wrap(err, "base")
	}
	if err := ValidateAsset(quoteType, quote); err != nil {
		return errorsmod.Wrap(err, "quote")
	}
	if baseType == quoteType && base == quote {
		return ErrIdenticalPair
	}
	return nil
}

func validateRate(rate uint64, rateScale uint32, validity time.Duration) error {
	if rate == 0 {
		return errorsmod.Wrap(ErrInvalidRate, "rate must be greater than 0")
	}
	if rateScale > 18 {
		return ErrInvalidRateScale
	}
	if validity < minValidityDuration || !storable(validity) {
		return errorsmod.Wrapf(ErrInvalidDuration, "got %s", validity)
	}
	return nil
}

// validateBounds checks the optional authorization bounds; zero is unset.
func validateBounds(minInterval time.Duration, bps uint32) error {
	if minInterval < 0 || !storable(minInterval) {
		return errorsmod.Wrapf(ErrInvalidMinInterval, "got %s", minInterval)
	}
	if bps > maxDeviationBps {
		return ErrInvalidMaxDeviation
	}
	return nil
}

// storable reports whether d is a whole number of seconds that fits uint32.
func storable(d time.Duration) bool {
	return d%time.Second == 0 && d <= maxDuration
}
