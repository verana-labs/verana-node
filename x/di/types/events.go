package types

const (
	EventTypeStoreDigest = "store_digest"

	AttributeKeyCorporation = "corporation"
	AttributeKeyOperator    = "operator"
	AttributeKeyDigest      = "digest"
	AttributeKeySource      = "source"
	AttributeKeyTimestamp   = "timestamp"

	// AttributeValueSourceMsg / SourceModuleCall distinguish the two entry points.
	AttributeValueSourceMsg        = "msg"
	AttributeValueSourceModuleCall = "module_call"
)
