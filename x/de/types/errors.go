package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/de module sentinel errors
var (
	ErrInvalidSigner               = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidMsgType              = errors.Register(ModuleName, 1101, "invalid or non-delegable message type")
	ErrExpirationInPast            = errors.Register(ModuleName, 1102, "expiration must be in the future")
	ErrVSOperatorAuthzExists       = errors.Register(ModuleName, 1103, "VSOperatorAuthorization already exists for this corporation/grantee pair; mutual exclusivity violated")
	ErrInvalidSpendLimit           = errors.Register(ModuleName, 1104, "invalid spend limit")
	ErrAuthzNotFound               = errors.Register(ModuleName, 1105, "delegate authorization not found for this corporation/delegate pair")
	ErrAuthzExpired                = errors.Register(ModuleName, 1106, "delegate authorization has expired")
	ErrAuthzMsgTypeNotFound        = errors.Register(ModuleName, 1107, "delegate authorization does not include requested message type")
	ErrAuthzSpendLimitExceeded     = errors.Register(ModuleName, 1108, "delegate authorization spend limit exceeded")
	ErrDelegateAuthzNotFound       = errors.Register(ModuleName, 1109, "delegate authorization not found for this corporation/grantee pair")
	ErrDelegateAuthzExistsMutex    = errors.Register(ModuleName, 1110, "DelegateAuthorization already exists for this corporation/vs_operator pair; mutual exclusivity violated")
	ErrInvalidResponseMaxSize      = errors.Register(ModuleName, 1111, "response_max_size must be between 1 and 1024")
	ErrParticipantRecordExists     = errors.Register(ModuleName, 1112, "a ParticipantAuthorizationRecord already exists for this participant_id; must be globally unique")
	ErrVSOAOtherCorporation        = errors.Register(ModuleName, 1113, "vs_operator already has a VSOperatorAuthorization from a different corporation; single-corp constraint violated")
	ErrVSOperatorAuthzNotFound     = errors.Register(ModuleName, 1114, "VS operator authorization not found")
	ErrVSOFeegrantNotEnabled       = errors.Register(ModuleName, 1115, "VS operator authorization record does not enable fee grant")
	ErrParticipantNotActive        = errors.Register(ModuleName, 1116, "participant is not an active participant")
	ErrInvalidFeeSpendLimit        = errors.Register(ModuleName, 1117, "fee_spend_limit must be set and strictly positive iff with_feegrant is true")
	ErrOwnPolicyAddress            = errors.Register(ModuleName, 1118, "grantee or vs_operator must not be the corporation's own policy_address")
	ErrAccreditationNotActive      = errors.Register(ModuleName, 1119, "primary entry is not an active accreditation")
	ErrVSAccountNotAuthorized      = errors.Register(ModuleName, 1120, "signer is not an authorized VS account of the corporation's service entry")
	ErrVSAccountFeeGrantNotEnabled = errors.Register(ModuleName, 1121, "service entry has an empty fee_spend_limit: the corporation does not pay this VS account's fees")
	ErrFeeAllowanceNotOwned        = errors.Register(ModuleName, 1122, "the corporation's fee allowance to this grantee was not created by this module")
)
