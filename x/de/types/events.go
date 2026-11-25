package types

const (
	EventTypeGrantDelegateAuthorization    = "grant_delegate_authorization"
	EventTypeRevokeDelegateAuthorization   = "revoke_delegate_authorization"
	EventTypeGrantFeeAllowance             = "grant_fee_allowance"
	EventTypeRevokeFeeAllowance            = "revoke_fee_allowance"
	EventTypeGrantVSOperatorAuthorization  = "grant_vs_operator_authorization"
	EventTypeRevokeVSOperatorAuthorization = "revoke_vs_operator_authorization"
	EventTypeUpdateVSOperatorAuthorization = "update_vs_operator_authorization"

	// Emitted whenever an AUTHZ-CHECK path mutates an authorization between grant
	// and revoke (spend debit or cycle renewal). Carries only ids: the indexer
	// re-reads authoritative state via ABCI at that height.
	EventTypeDelegateAuthorizationUpdated   = "delegate_authorization_updated"
	EventTypeVSOperatorAuthorizationUpdated = "vs_operator_authorization_updated"

	AttributeKeyCorporation   = "corporation"
	AttributeKeyCorporationID = "corporation_id"
	AttributeKeyGrantee       = "grantee"
	AttributeKeyWithFeegrant  = "with_feegrant"
	AttributeKeyTimestamp     = "timestamp"
	AttributeKeyVsOperator    = "vs_operator"
	AttributeKeyPermissionID  = "permission_id"
	AttributeKeyAuthzID       = "authz_id"
	AttributeKeyVsoaID        = "vsoa_id"
	AttributeKeyParticipantID = "participant_id"
)
