package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/ds module sentinel errors
var (
	ErrInvalidSigner = errors.Register(ModuleName, 1100, "expected council account as only signer for proposal message")
	ErrInvalidParams = errors.Register(ModuleName, 1101, "invalid parameters")
)
