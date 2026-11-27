package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                     DefaultParams(),
		ExchangeRates:              []ExchangeRate{},
		NextExchangeRateId:         0,
		ExchangeRateAuthorizations: []ExchangeRateAuthorization{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	rateIDs := make(map[uint64]struct{}, len(gs.ExchangeRates))
	pairs := make(map[string]struct{}, len(gs.ExchangeRates))
	var maxID uint64
	for i := range gs.ExchangeRates {
		xr := &gs.ExchangeRates[i]
		if xr.Id == 0 {
			return fmt.Errorf("exchange rate has zero id")
		}
		if _, dup := rateIDs[xr.Id]; dup {
			return fmt.Errorf("duplicate exchange rate id %d", xr.Id)
		}
		rateIDs[xr.Id] = struct{}{}
		if err := validatePair(xr.BaseAssetType, xr.BaseAsset, xr.QuoteAssetType, xr.QuoteAsset); err != nil {
			return fmt.Errorf("exchange rate %d: %w", xr.Id, err)
		}
		if err := validateRate(xr.Rate, xr.RateScale, xr.ValidityDuration); err != nil {
			return fmt.Errorf("exchange rate %d: %w", xr.Id, err)
		}
		pairKey := fmt.Sprintf("%d|%s|%d|%s", xr.BaseAssetType, xr.BaseAsset, xr.QuoteAssetType, xr.QuoteAsset)
		if _, dup := pairs[pairKey]; dup {
			return fmt.Errorf("duplicate exchange rate pair %s", pairKey)
		}
		pairs[pairKey] = struct{}{}
		maxID = max(maxID, xr.Id)
	}
	if gs.NextExchangeRateId < maxID {
		return fmt.Errorf("next_exchange_rate_id %d is less than the highest exchange rate id %d", gs.NextExchangeRateId, maxID)
	}

	seen := make(map[string]struct{}, len(gs.ExchangeRateAuthorizations))
	for i := range gs.ExchangeRateAuthorizations {
		a := &gs.ExchangeRateAuthorizations[i]
		if _, ok := rateIDs[a.XrId]; !ok {
			return fmt.Errorf("exchange rate authorization references unknown xr_id %d", a.XrId)
		}
		feeder, err := sdk.AccAddressFromBech32(a.Feeder)
		if err != nil {
			return fmt.Errorf("exchange rate authorization for xr_id %d has invalid feeder: %w", a.XrId, err)
		}
		if a.Expiration.IsZero() {
			return fmt.Errorf("exchange rate authorization (xr_id %d, feeder %s) has no expiration", a.XrId, a.Feeder)
		}
		if err := validateBounds(a.MinInterval, a.MaxDeviationBps); err != nil {
			return fmt.Errorf("exchange rate authorization (xr_id %d, feeder %s): %w", a.XrId, a.Feeder, err)
		}
		key := fmt.Sprintf("%d/%s", a.XrId, feeder)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate exchange rate authorization for (xr_id %d, feeder %s)", a.XrId, a.Feeder)
		}
		seen[key] = struct{}{}
	}

	return nil
}
