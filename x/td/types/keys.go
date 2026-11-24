package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "td"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_trustdeposit"

	RouterKey             = ModuleName
	YieldIntermediatePool = "yield_intermediate_pool"
)

var (
	ParamsKey       = []byte("p_trustdeposit")
	TrustDepositKey = collections.NewPrefix(1)
	DustKey         = collections.NewPrefix(2)
	// PendingTuDecayRateKey holds the tu_decay_rate applied at the next epoch boundary.
	PendingTuDecayRateKey = collections.NewPrefix(3)
	// ShareValueKey holds the v4 trust deposit share value.
	ShareValueKey = collections.NewPrefix(4)
)

const (
	BondDenom = "uvna"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
