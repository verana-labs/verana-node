package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "di"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

// DigestsKey is the prefix for the Digests collection.
var DigestsKey = collections.NewPrefix("d_di")
