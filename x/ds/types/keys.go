package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "ds"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// FoundationPlaceholder names the module address used as foundation_account
	// in the default genesis, until the genesis state defines the real account.
	FoundationPlaceholder = "foundation"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_ds")
