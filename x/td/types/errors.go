package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/td module sentinel errors
var (
	ErrInvalidSigner            = errors.Register(ModuleName, 1100, "expected council account as only signer for proposal message")
	ErrSample                   = errors.Register(ModuleName, 1101, "sample error")
	ErrInvalidAmount            = errors.Register(ModuleName, 1103, "invalid amount")
	ErrTrustDepositNotFound     = errors.Register(ModuleName, 1104, "trust deposit not found")
	ErrInsufficientTrustDeposit = errors.Register(ModuleName, 1105, "insufficient trust deposit")
	ErrMsgVoid                  = errors.Register(ModuleName, 1106, "message is void in v5")
)
