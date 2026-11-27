package types

const (
	EventTypeCreateExchangeRate              = "create_exchange_rate"
	EventTypeUpdateExchangeRate              = "update_exchange_rate"
	EventTypeSetExchangeRateState            = "set_exchange_rate_state"
	EventTypeGrantExchangeRateAuthorization  = "grant_exchange_rate_authorization"
	EventTypeRevokeExchangeRateAuthorization = "revoke_exchange_rate_authorization"
	EventTypeUpdateModuleParameters          = "update_module_parameters"

	AttributeKeyXrID    = "xr_id"
	AttributeKeyFeeder  = "feeder"
	AttributeKeyUpdated = "updated"
	AttributeKeyExpires = "expires"
)
