package types

const (
	EventTypeAddGovernanceFrameworkDocument           = "add_governance_framework_document"
	EventTypeIncreaseActiveGovernanceFrameworkVersion = "increase_active_governance_framework_version"
	EventTypeAcceptGovernanceFrameworkVersion         = "accept_governance_framework_version"
	EventTypeAcceptCorporationGovernanceFramework     = "accept_corporation_governance_framework"
	EventTypeUpdateModuleParameters                   = "update_module_parameters"

	AttributeKeyGFVersionID        = "gfv_id"
	AttributeKeyGFDocID            = "gfd_id"
	AttributeKeyVersion            = "version"
	AttributeKeyNetwork            = "network"
	AttributeKeySigner             = "signer"
	AttributeKeyActiveSince        = "active_since"
	AttributeKeyAcceptanceDeadline = "acceptance_deadline"
)
