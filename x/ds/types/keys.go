package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name; its module account is the distribution pool.
	ModuleName = "ds"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// FoundationPlaceholder names the module address used as foundation_account
	// in the default genesis, until the genesis state defines the real account.
	FoundationPlaceholder = "foundation"
)

var (
	// ParamsKey is the prefix to retrieve all Params
	ParamsKey = collections.NewPrefix("p_ds")
	// ValidatorBlocksKey holds the open epoch's counters by operator address.
	ValidatorBlocksKey = collections.NewPrefix(1)
	// EpochBlocksKey holds the number of blocks of the open epoch.
	EpochBlocksKey = collections.NewPrefix(2)
)
