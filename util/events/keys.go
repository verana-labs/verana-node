package events

import sdk "github.com/cosmos/cosmos-sdk/types"

// Attribute keys shared by several modules.
const (
	AttrModule          = sdk.AttributeKeyModule
	AttrNow             = "now"
	AttrMsg             = "msg"
	AttrCorporationID   = "corporation_id"
	AttrEcosystemID     = "ecosystem_id"
	AttrVsID            = "vs_id"
	AttrSchemaID        = "schema_id"
	AttrRole            = "role"
	AttrAccreditationID = "accreditation_id"
)
