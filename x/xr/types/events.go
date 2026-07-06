package types

const (
	EventTypeCreateExchangeRate      = "create_exchange_rate"
	EventTypeUpdateExchangeRate      = "update_exchange_rate"
	EventTypeSetExchangeRateState    = "set_exchange_rate_state"
	EventTypeGrantExchangeRateAuthz  = "grant_exchange_rate_authz"
	EventTypeRevokeExchangeRateAuthz = "revoke_exchange_rate_authz"

	AttributeKeyID               = "id"
	AttributeKeyBaseAssetType    = "base_asset_type"
	AttributeKeyBaseAsset        = "base_asset"
	AttributeKeyQuoteAssetType   = "quote_asset_type"
	AttributeKeyQuoteAsset       = "quote_asset"
	AttributeKeyAuthority        = "authority"
	AttributeKeyOperator         = "operator"
	AttributeKeyRate             = "rate"
	AttributeKeyState            = "state"
	AttributeKeyRateScale        = "rate_scale"
	AttributeKeyValidityDuration = "validity_duration"
	AttributeKeyExpires          = "expires"
	AttributeKeyExpiration       = "expiration"
	AttributeKeyMinInterval      = "min_interval"
	AttributeKeyMaxDeviationBps  = "max_deviation_bps"
)
