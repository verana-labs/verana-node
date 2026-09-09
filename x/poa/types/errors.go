package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidSigner          = errorsmod.Register(ModuleName, 2, "expected council authority as signer")
	ErrNotCouncilMember       = errorsmod.Register(ModuleName, 3, "address is not a council member")
	ErrMaxValidatorsReached   = errorsmod.Register(ModuleName, 4, "max validators reached")
	ErrAddressHasBondedTokens = errorsmod.Register(ModuleName, 5, "address already has a validator with bonded tokens")
	ErrAddressHasDelegations  = errorsmod.Register(ModuleName, 6, "address has delegations")
	ErrNotAValidator          = errorsmod.Register(ModuleName, 7, "address is not a validator")
	ErrInvalidValidatorStatus = errorsmod.Register(ModuleName, 8, "invalid validator status")
	ErrMsgNotAllowed          = errorsmod.Register(ModuleName, 9, "staking message not allowed under proof of authority")
	ErrInvalidBond            = errorsmod.Register(ModuleName, 10, "genesis validator bond must equal the fixed council bond")
	ErrLastValidator          = errorsmod.Register(ModuleName, 11, "cannot remove the last bonded validator")
	ErrCouncilIntegrity       = errorsmod.Register(ModuleName, 12, "message would break the council's self-administration")
)
