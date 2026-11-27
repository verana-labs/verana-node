package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "vs"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_vs")
