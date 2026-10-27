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
	// RootParticipantBySchemaCorpKey indexes (schema_id, corporation_id, participant_id) for ECOSYSTEM entries.
	RootParticipantBySchemaCorpKey = collections.NewPrefix(4)
	// UnrepaidSlashByCorpKey indexes (corporation_id, schema_id, participant_id) while a slash is unrepaid.
	UnrepaidSlashByCorpKey = collections.NewPrefix(5)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
