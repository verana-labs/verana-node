package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "pp"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_pp"
)

const (
	BondDenom = "uvna"
)

var (
	ParamsKey             = []byte("p_pp")
	ParticipantKey        = collections.NewPrefix(0)
	ParticipantCounterKey = collections.NewPrefix(1)
	ParticipantSessionKey = collections.NewPrefix(2)
	// ParticipantByDIDCorpKey indexes (did, participant_id) -> corporation_id so a
	// participant's owning corporation can be resolved by DID in O(log n) instead
	// of a linear Participant walk. Mirrors x/ec EcosystemByDIDCorpKey.
	ParticipantByDIDCorpKey = collections.NewPrefix(3)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
