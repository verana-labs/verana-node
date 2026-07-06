package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/verana-labs/verana-node/util/validation"
)

// ValidateDigestString checks a digest is non-empty, within the 256-byte cap,
// and a valid SRI string. Shared by the Msg and module-call paths.
func ValidateDigestString(digest string) error {
	if digest == "" {
		return ErrDigestEmpty
	}
	if len(digest) > 256 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest exceeds maximum length of 256 bytes")
	}
	// [MOD-DI-MSG-1-1] digest must be a valid SRI string per spec v4 draft 13.
	if !validation.IsValidDigestSRI(digest) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest must be a valid SRI string (e.g. sha256-<base64>)")
	}
	return nil
}

// ValidateBasic performs stateless validation on MsgStoreDigest.
func (msg *MsgStoreDigest) ValidateBasic() error {
	// authority (corporation) is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// operator is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	return ValidateDigestString(msg.Digest)
}
