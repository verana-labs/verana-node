package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/ec sentinel errors. Codes are arbitrary but stable within this module.
var (
	ErrInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected council account as authority for proposal message")
	ErrEcosystemNotFound           = sdkerrors.Register(ModuleName, 1101, "ecosystem not found")
	ErrUnauthorizedDelegate        = sdkerrors.Register(ModuleName, 1103, "signing corporation does not control this ecosystem")
	ErrDIDOwnershipConflict        = sdkerrors.Register(ModuleName, 1104, "did already bound to an ecosystem controlled by a different corporation")
	ErrAlreadyInTargetArchiveState = sdkerrors.Register(ModuleName, 1105, "ecosystem already in target archive state")
	ErrInvalidDID                  = sdkerrors.Register(ModuleName, 1106, "invalid DID")
	ErrInvalidLanguage             = sdkerrors.Register(ModuleName, 1107, "invalid language tag")
	ErrInvalidURL                  = sdkerrors.Register(ModuleName, 1108, "invalid URL")
	ErrInvalidDigestSRI            = sdkerrors.Register(ModuleName, 1109, "invalid digest_sri")
	ErrInvalidSubject              = sdkerrors.Register(ModuleName, 1110, "invalid subject id")
	ErrInvalidTimestamp            = sdkerrors.Register(ModuleName, 1111, "invalid ecosystem timestamp")
	ErrNoParams                    = sdkerrors.Register(ModuleName, 1112, "the module owns no parameter")
)
