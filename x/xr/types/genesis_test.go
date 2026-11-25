package types_test

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

func validGenesis() *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.ExchangeRates = []types.ExchangeRate{{
		Id:               1,
		BaseAssetType:    cstypes.PricingAssetType_FIAT,
		BaseAsset:        "USD",
		QuoteAssetType:   cstypes.PricingAssetType_COIN,
		QuoteAsset:       "uvna",
		Rate:             1_000_000,
		RateScale:        0,
		ValidityDuration: 24 * time.Hour,
		Updated:          time.Unix(1_700_000_000, 0).UTC(),
		Expires:          time.Unix(1_700_086_400, 0).UTC(),
		State:            true,
	}}
	gs.NextExchangeRateId = 1
	gs.ExchangeRateAuthorizations = []types.ExchangeRateAuthorization{{
		XrId:            1,
		Feeder:          sdk.AccAddress([]byte("xr_genesis_feeder___")).String(),
		Expiration:      time.Unix(1_800_000_000, 0).UTC(),
		MinInterval:     time.Minute,
		MaxDeviationBps: 1000,
	}}
	return gs
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc   string
		mutate func(*types.GenesisState)
		valid  bool
	}{
		{"default is valid", func(gs *types.GenesisState) { *gs = *types.DefaultGenesis() }, true},
		{"empty params is invalid (zero twap_window)", func(gs *types.GenesisState) { *gs = types.GenesisState{} }, false},
		{"entry and authorization", func(*types.GenesisState) {}, true},
		{"trust unit asset", func(gs *types.GenesisState) {
			gs.ExchangeRates[0].BaseAssetType = cstypes.PricingAssetType_TU
			gs.ExchangeRates[0].BaseAsset = "tu"
		}, false},
		{"invalid fiat code", func(gs *types.GenesisState) { gs.ExchangeRates[0].BaseAsset = "usd" }, false},
		{"invalid denom", func(gs *types.GenesisState) { gs.ExchangeRates[0].QuoteAsset = "1" }, false},
		{"identical pair", func(gs *types.GenesisState) {
			gs.ExchangeRates[0].QuoteAssetType = cstypes.PricingAssetType_FIAT
			gs.ExchangeRates[0].QuoteAsset = "USD"
		}, false},
		{"zero rate", func(gs *types.GenesisState) { gs.ExchangeRates[0].Rate = 0 }, false},
		{"rate_scale above 18", func(gs *types.GenesisState) { gs.ExchangeRates[0].RateScale = 19 }, false},
		{"validity below one minute", func(gs *types.GenesisState) { gs.ExchangeRates[0].ValidityDuration = 59 * time.Second }, false},
		{"zero id", func(gs *types.GenesisState) {
			gs.ExchangeRates[0].Id = 0
			gs.ExchangeRateAuthorizations = nil
		}, false},
		{"duplicate id", func(gs *types.GenesisState) {
			second := gs.ExchangeRates[0]
			second.BaseAsset = "EUR"
			gs.ExchangeRates = append(gs.ExchangeRates, second)
		}, false},
		{"duplicate pair", func(gs *types.GenesisState) {
			second := gs.ExchangeRates[0]
			second.Id = 2
			gs.ExchangeRates = append(gs.ExchangeRates, second)
			gs.NextExchangeRateId = 2
		}, false},
		{"next id below highest id", func(gs *types.GenesisState) { gs.NextExchangeRateId = 0 }, false},
		{"authorization of unknown entry", func(gs *types.GenesisState) { gs.ExchangeRateAuthorizations[0].XrId = 2 }, false},
		{"invalid feeder", func(gs *types.GenesisState) { gs.ExchangeRateAuthorizations[0].Feeder = "feeder" }, false},
		{"no expiration", func(gs *types.GenesisState) { gs.ExchangeRateAuthorizations[0].Expiration = time.Time{} }, false},
		{"negative min_interval", func(gs *types.GenesisState) { gs.ExchangeRateAuthorizations[0].MinInterval = -time.Second }, false},
		{"sub-second min_interval", func(gs *types.GenesisState) { gs.ExchangeRateAuthorizations[0].MinInterval = time.Second / 2 }, false},
		{"fractional validity", func(gs *types.GenesisState) { gs.ExchangeRates[0].ValidityDuration = time.Hour + time.Millisecond }, false},
		{"unset min_interval and max_deviation_bps", func(gs *types.GenesisState) {
			gs.ExchangeRateAuthorizations[0].MinInterval = 0
			gs.ExchangeRateAuthorizations[0].MaxDeviationBps = 0
		}, true},
		{"max_deviation_bps above 10000", func(gs *types.GenesisState) { gs.ExchangeRateAuthorizations[0].MaxDeviationBps = 10001 }, false},
		{"duplicate authorization", func(gs *types.GenesisState) {
			gs.ExchangeRateAuthorizations = append(gs.ExchangeRateAuthorizations, gs.ExchangeRateAuthorizations[0])
		}, false},
		{"duplicate authorization with upper-case feeder", func(gs *types.GenesisState) {
			second := gs.ExchangeRateAuthorizations[0]
			second.Feeder = strings.ToUpper(second.Feeder)
			gs.ExchangeRateAuthorizations = append(gs.ExchangeRateAuthorizations, second)
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			gs := validGenesis()
			tc.mutate(gs)
			err := gs.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
