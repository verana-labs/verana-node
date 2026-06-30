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

	seen := make(map[string]struct{}, len(gs.ExchangeRateAuthorizations))
	for _, a := range gs.ExchangeRateAuthorizations {
		if a.XrId == 0 {
			return fmt.Errorf("exchange rate authorization has zero xr_id")
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
