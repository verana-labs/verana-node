package types

import (
	"fmt"

	"cosmossdk.io/math"
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
	for _, xr := range gs.ExchangeRates {
		if xr.Id == 0 {
			return fmt.Errorf("exchange rate has zero id")
		}
		if _, dup := rateIDs[xr.Id]; dup {
			return fmt.Errorf("duplicate exchange rate id %d", xr.Id)
		}
		rateIDs[xr.Id] = struct{}{}
		// [MOD-XR-MSG-1-2-1] rate is a base-10 unsigned integer string, not a decimal.
		if rate, ok := math.NewIntFromString(xr.Rate); !ok || !rate.IsPositive() {
			return fmt.Errorf("exchange rate %d has invalid rate %q (must be a positive integer)", xr.Id, xr.Rate)
		}
		if xr.RateScale > 18 {
			return fmt.Errorf("exchange rate %d has rate_scale %d > 18", xr.Id, xr.RateScale)
		}
		pairKey := fmt.Sprintf("%d|%s|%d|%s", xr.BaseAssetType, xr.BaseAsset, xr.QuoteAssetType, xr.QuoteAsset)
		if _, dup := pairs[pairKey]; dup {
			return fmt.Errorf("duplicate exchange rate pair %s", pairKey)
		}
		pairs[pairKey] = struct{}{}
		if xr.Id > maxID {
			maxID = xr.Id
		}
	}
	if gs.NextExchangeRateId < maxID {
		return fmt.Errorf("next_exchange_rate_id %d is less than the highest exchange rate id %d", gs.NextExchangeRateId, maxID)
	}

	seen := make(map[string]struct{}, len(gs.ExchangeRateAuthorizations))
	for _, a := range gs.ExchangeRateAuthorizations {
		if a.XrId == 0 {
			return fmt.Errorf("exchange rate authorization has zero xr_id")
		}
		if _, ok := rateIDs[a.XrId]; !ok {
			return fmt.Errorf("exchange rate authorization references unknown xr_id %d", a.XrId)
		}
		if _, err := sdk.AccAddressFromBech32(a.Operator); err != nil {
			return fmt.Errorf("exchange rate authorization for xr_id %d has invalid operator: %w", a.XrId, err)
		}
		if a.Expiration.IsZero() {
			return fmt.Errorf("exchange rate authorization (xr_id %d, operator %s) has no expiration", a.XrId, a.Operator)
		}
		if a.MaxDeviationBps > 10000 {
			return fmt.Errorf("exchange rate authorization (xr_id %d, operator %s) max_deviation_bps %d exceeds 10000", a.XrId, a.Operator, a.MaxDeviationBps)
		}
		key := fmt.Sprintf("%d/%s", a.XrId, a.Operator)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate exchange rate authorization for (xr_id %d, operator %s)", a.XrId, a.Operator)
		}
		seen[key] = struct{}{}
	}

	return nil
}
