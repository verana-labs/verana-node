package types

const (
	EventTypeStoreDigest = "store_digest"

	AttributeKeyCorporation = "corporation"
	AttributeKeyDelegate    = "delegate"
	AttributeKeyDigest      = "digest"
	AttributeKeySource      = "source"
	AttributeKeyTimestamp   = "timestamp"

	// AttributeValueSourceMsg / SourceModuleCall distinguish the two entry points.
	AttributeValueSourceMsg        = "msg"
	AttributeValueSourceModuleCall = "module_call"
)
