package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "xr"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// CounterKeyExchangeRate is the counter key for exchange rate auto-id
	CounterKeyExchangeRate = "exchange_rate"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_xr")

// ExchangeRateKey is the prefix for exchange rate storage
var ExchangeRateKey = collections.NewPrefix("xr_rate")

// CounterKey is the prefix for counters
var CounterKey = collections.NewPrefix("xr_ctr")

// ExchangeRateAuthorizationKey is the prefix for exchange rate authorizations,
// keyed by (xr_id, feeder).
var ExchangeRateAuthorizationKey = collections.NewPrefix("xr_authz")
